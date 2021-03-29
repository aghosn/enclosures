#include <stddef.h>
#include <sys/types.h>
#include <assert.h>
#include <stdint.h>
#include <stdlib.h>
#include <stdio.h>
#include <sys/mman.h>
#include <stdbool.h>
#include <string.h>

#include "pymacro.h"
#include "mtypes.h"
#include "mh_interface.h"
#include "mh_state.h"
#include "mh_api.h"

// We will use the thing from python, see if it works.
extern void* PyMem_RawRealloc(void *ptr, size_t new_size);
extern void PyMem_RawFree(void *ptr);
extern void * PyMem_RawCalloc(size_t nelem, size_t elsize);
extern void * PyMem_RawMalloc(size_t size);
extern Py_ssize_t _Py_GetAllocatedBlocks(void);

static void *
_PyObject_ArenaMmap(void *ctx, size_t size)
{
    void *ptr;
    ptr = mmap(NULL, size, PROT_READ|PROT_WRITE,
               MAP_PRIVATE|MAP_ANONYMOUS, -1, 0);
    if (ptr == MAP_FAILED)
        return NULL;
    assert(ptr != NULL);
    //Catch the mmap and register the result.
    if (register_growth != NULL) {
      int64_t id = mh_stack_peek();
      register_growth(id, ptr, size);
    }
    return ptr;
}

static void
_PyObject_ArenaMunmap(void *ctx, void *ptr, size_t size)
{
    munmap(ptr, size);
}

static ObjectArenaAllocator _PyObject_Arena = {NULL,
    _PyObject_ArenaMmap, _PyObject_ArenaMunmap
};

/* Allocate a new arena.  If we run out of memory, return NULL.  Else
 * allocate a new arena, and return the address of an arena_object
 * describing the new arena.  It's expected that the caller will set
 * `mhcurr->usable_arenas` to the return value.
 */
static struct arena_object*
new_arena(mh_state* mhcurr)
{
    struct arena_object* arenaobj;
    uint excess;        /* number of bytes above pool alignment */
    void *address;
    static int debug_stats = -1;

    if (debug_stats == -1) {
        const char *opt = MHPy_GETENV("PYTHONMALLOCSTATS");
        debug_stats = (opt != NULL && *opt != '\0');
    }

    if (mhcurr->unused_arena_objects == NULL) {
        uint i;
        uint numarenas;
        size_t nbytes;

        /* Double the number of arena objects on each allocation.
         * Note that it's possible for `numarenas` to overflow.
         */
        numarenas = mhcurr->maxarenas ? mhcurr->maxarenas << 1 : INITIAL_ARENA_OBJECTS;
        if (numarenas <= mhcurr->maxarenas)
            return NULL;                /* overflow */
#if SIZEOF_SIZE_T <= SIZEOF_INT
        if (numarenas > SIZE_MAX / sizeof(*(mhcurr->arenas)))
            return NULL;                /* overflow */
#endif
        nbytes = numarenas * sizeof(*(mhcurr->arenas));
        arenaobj = (struct arena_object *)PyMem_RawRealloc(mhcurr->arenas, nbytes);
        if (arenaobj == NULL)
            return NULL;
        mhcurr->arenas = arenaobj;

        /* We might need to fix pointers that were copied.  However,
         * new_arena only gets called when all the pages in the
         * previous arenas are full.  Thus, there are *no* pointers
         * into the old array. Thus, we don't have to worry about
         * invalid pointers.  Just to be sure, some asserts:
         */
        assert(mhcurr->usable_arenas == NULL);
        assert(mhcurr->unused_arena_objects == NULL);

        /* Put the new arenas on the mhcurr->unused_arena_objects list. */
        for (i = mhcurr->maxarenas; i < numarenas; ++i) {
            mhcurr->arenas[i].address = 0;              /* mark as unassociated */
            mhcurr->arenas[i].nextarena = i < numarenas - 1 ?
                                   &mhcurr->arenas[i+1] : NULL;
        }

        /* Update globals. */
        mhcurr->unused_arena_objects = &(mhcurr->arenas[mhcurr->maxarenas]);
        mhcurr->maxarenas = numarenas;
    }

    /* Take the next available arena object off the head of the list. */
    assert(mhcurr->unused_arena_objects != NULL);
    arenaobj = mhcurr->unused_arena_objects;
    mhcurr->unused_arena_objects = arenaobj->nextarena;
    assert(arenaobj->address == 0);
    /* Push the pool_id before the call so we get access to it in mmap. */
    mh_stack_push(mhcurr->pool_id);
    address = _PyObject_Arena.alloc(_PyObject_Arena.ctx, ARENA_SIZE);
    assert(address != NULL && (mhcurr->pool_id == mh_stack_pop()));
    if (address == NULL) {
        /* The allocation failed: return NULL after putting the
         * arenaobj back.
         */
        arenaobj->nextarena = mhcurr->unused_arena_objects;
        mhcurr->unused_arena_objects = arenaobj;
        return NULL;
    }
    arenaobj->address = (uintptr_t)address;

    ++(mhcurr->narenas_currently_allocated);
    ++(mhcurr->ntimes_arena_allocated);
    if (mhcurr->narenas_currently_allocated > mhcurr->narenas_highwater)
        mhcurr->narenas_highwater = mhcurr->narenas_currently_allocated;
    arenaobj->freepools = NULL;
    /* pool_address <- first pool-aligned address in the arena
       nfreepools <- number of whole pools that fit after alignment */
    arenaobj->pool_address = (block*)arenaobj->address;
    arenaobj->nfreepools = MAX_POOLS_IN_ARENA;
    excess = (uint)(arenaobj->address & POOL_SIZE_MASK);
    if (excess != 0) {
        --(arenaobj->nfreepools);
        arenaobj->pool_address += POOL_SIZE - excess;
    }
    arenaobj->ntotalpools = arenaobj->nfreepools;

    return arenaobj;
}

static bool _Py_NO_SANITIZE_ADDRESS
            _Py_NO_SANITIZE_THREAD
            _Py_NO_SANITIZE_MEMORY
address_in_range(mh_state* mhcurr, void *p, poolp pool)
{
    // Since address_in_range may be reading from memory which was not allocated
    // by Python, it is important that pool->arenaindex is read only once, as
    // another thread may be concurrently modifying the value without holding
    // the GIL. The following dance forces the compiler to read pool->arenaindex
    // only once.
    uint arenaindex = *((volatile uint *)&pool->arenaindex);
    return arenaindex < mhcurr->maxarenas &&
        (uintptr_t)p - (mhcurr->arenas[arenaindex].address) < ARENA_SIZE &&
        mhcurr->arenas[arenaindex].address != 0;
}


/*==========================================================================*/

// Called when freelist is exhausted.  Extend the freelist if there is
// space for a block.  Otherwise, remove this pool from usedpools.
static void
pymalloc_pool_extend(poolp pool, uint size)
{
    if (UNLIKELY(pool->nextoffset <= pool->maxnextoffset)) {
        /* There is room for another block. */
        pool->freeblock = (block*)pool + pool->nextoffset;
        pool->nextoffset += INDEX2SIZE(size);
        *(block **)(pool->freeblock) = NULL;
        return;
    }

    /* Pool is full, unlink from used pools. */
    poolp next;
    next = pool->nextpool;
    pool = pool->prevpool;
    next->prevpool = pool;
    pool->nextpool = next;
}

/* called when pymalloc_alloc can not allocate a block from usedpool.
 * This function takes new pool and allocate a block from it.
 */
static void*
allocate_from_new_pool(mh_state* mhcurr, uint size)
{
    /* There isn't a pool of the right size class immediately
     * available:  use a free pool.
     */
    if (UNLIKELY(mhcurr->usable_arenas == NULL)) {
        /* No arena has a free pool:  allocate a new arena. */
#ifdef WITH_MEMORY_LIMITS
        if (mhcurr->narenas_currently_allocated >= MAX_ARENAS) {
            return NULL;
        }
#endif
        mhcurr->usable_arenas = new_arena(mhcurr);
        if (mhcurr->usable_arenas == NULL) {
            return NULL;
        }
        mhcurr->usable_arenas->nextarena = mhcurr->usable_arenas->prevarena = NULL;
        assert(mhcurr->nfp2lasta[mhcurr->usable_arenas->nfreepools] == NULL);
        mhcurr->nfp2lasta[mhcurr->usable_arenas->nfreepools] = mhcurr->usable_arenas;
    }
    assert(mhcurr->usable_arenas->address != 0);

    /* This arena already had the smallest nfreepools value, so decreasing
     * nfreepools doesn't change that, and we don't need to rearrange the
     * mhcurr->usable_arenas list.  However, if the arena becomes wholly allocated,
     * we need to remove its arena_object from mhcurr->usable_arenas.
     */
    assert(mhcurr->usable_arenas->nfreepools > 0);
    if (mhcurr->nfp2lasta[mhcurr->usable_arenas->nfreepools] == mhcurr->usable_arenas) {
        /* It's the last of this size, so there won't be any. */
        mhcurr->nfp2lasta[mhcurr->usable_arenas->nfreepools] = NULL;
    }
    /* If any free pools will remain, it will be the new smallest. */
    if (mhcurr->usable_arenas->nfreepools > 1) {
        assert(mhcurr->nfp2lasta[mhcurr->usable_arenas->nfreepools - 1] == NULL);
        mhcurr->nfp2lasta[mhcurr->usable_arenas->nfreepools - 1] = mhcurr->usable_arenas;
    }

    /* Try to get a cached free pool. */
    poolp pool = mhcurr->usable_arenas->freepools;
    if (LIKELY(pool != NULL)) {
        /* Unlink from cached pools. */
        mhcurr->usable_arenas->freepools = pool->nextpool;
        mhcurr->usable_arenas->nfreepools--;
        if (UNLIKELY(mhcurr->usable_arenas->nfreepools == 0)) {
            /* Wholly allocated:  remove. */
            assert(mhcurr->usable_arenas->freepools == NULL);
            assert(mhcurr->usable_arenas->nextarena == NULL ||
                   mhcurr->usable_arenas->nextarena->prevarena ==
                   mhcurr->usable_arenas);
            mhcurr->usable_arenas = mhcurr->usable_arenas->nextarena;
            if (mhcurr->usable_arenas != NULL) {
                mhcurr->usable_arenas->prevarena = NULL;
                assert(mhcurr->usable_arenas->address != 0);
            }
        }
        else {
            /* nfreepools > 0:  it must be that freepools
             * isn't NULL, or that we haven't yet carved
             * off all the arena's pools for the first
             * time.
             */
            assert(mhcurr->usable_arenas->freepools != NULL ||
                   mhcurr->usable_arenas->pool_address <=
                   (block*)mhcurr->usable_arenas->address +
                       ARENA_SIZE - POOL_SIZE);
        }
    }
    else {
        /* Carve off a new pool. */
        assert(mhcurr->usable_arenas->nfreepools > 0);
        assert(mhcurr->usable_arenas->freepools == NULL);
        pool = (poolp)mhcurr->usable_arenas->pool_address;
        assert((block*)pool <= (block*)mhcurr->usable_arenas->address +
                                 ARENA_SIZE - POOL_SIZE);
        pool->arenaindex = (uint)(mhcurr->usable_arenas - mhcurr->arenas);
        assert(&(mhcurr->arenas[pool->arenaindex]) == mhcurr->usable_arenas);
        pool->szidx = DUMMY_SIZE_IDX;
        mhcurr->usable_arenas->pool_address += POOL_SIZE;
        --mhcurr->usable_arenas->nfreepools;

        if (mhcurr->usable_arenas->nfreepools == 0) {
            assert(mhcurr->usable_arenas->nextarena == NULL ||
                   mhcurr->usable_arenas->nextarena->prevarena ==
                   mhcurr->usable_arenas);
            /* Unlink the arena:  it is completely allocated. */
            mhcurr->usable_arenas = mhcurr->usable_arenas->nextarena;
            if (mhcurr->usable_arenas != NULL) {
                mhcurr->usable_arenas->prevarena = NULL;
                assert(mhcurr->usable_arenas->address != 0);
            }
        }
    }

    /* Frontlink to used pools. */
    block *bp;
    poolp next = mhcurr->usedpools[size + size]; /* == prev */
    pool->nextpool = next;
    pool->prevpool = next;
    next->nextpool = pool;
    next->prevpool = pool;
    pool->ref.count = 1;
    if (pool->szidx == size) {
        /* Luckily, this pool last contained blocks
         * of the same size class, so its header
         * and free list are already initialized.
         */
        bp = pool->freeblock;
        assert(bp != NULL);
        pool->freeblock = *(block **)bp;
        return bp;
    }
    /*
     * Initialize the pool header, set up the free list to
     * contain just the second block, and return the first
     * block.
     */
    pool->szidx = size;
    size = INDEX2SIZE(size);
    bp = (block *)pool + POOL_OVERHEAD;
    pool->nextoffset = POOL_OVERHEAD + (size << 1);
    pool->maxnextoffset = POOL_SIZE - size;
    pool->freeblock = bp + size;
    *(block **)(pool->freeblock) = NULL;
    return bp;
}

/* pymalloc allocator

   Return a pointer to newly allocated memory if pymalloc allocated memory.

   Return NULL if pymalloc failed to allocate the memory block: on bigger
   requests, on error in the code below (as a last chance to serve the request)
   or when the max memory limit has been reached.
*/
static inline void*
pymalloc_alloc(mh_state* mhcurr, void *ctx, size_t nbytes)
{
#ifdef WITH_VALGRIND
    if (UNLIKELY(running_on_valgrind == -1)) {
        running_on_valgrind = RUNNING_ON_VALGRIND;
    }
    if (UNLIKELY(running_on_valgrind)) {
        return NULL;
    }
#endif

    if (UNLIKELY(nbytes == 0)) {
        return NULL;
    }
    if (UNLIKELY(nbytes > SMALL_REQUEST_THRESHOLD)) {
        return NULL;
    }

    uint size = (uint)(nbytes - 1) >> ALIGNMENT_SHIFT;
    poolp pool = mhcurr->usedpools[size + size];
    block *bp;

    if (LIKELY(pool != pool->nextpool)) {
        /*
         * There is a used pool for this size class.
         * Pick up the head block of its free list.
         */
        ++pool->ref.count;
        bp = pool->freeblock;
        assert(bp != NULL);

        if (UNLIKELY((pool->freeblock = *(block **)bp) == NULL)) {
            // Reached the end of the free list, try to extend it.
            pymalloc_pool_extend(pool, size);
        }
    }
    else {
        /* There isn't a pool of the right size class immediately
         * available:  use a free pool.
         */
        bp = allocate_from_new_pool(mhcurr, size);
    }

    return (void *)bp;
}

static void *
_Intrn_Malloc(mh_state* mhd_state, void *ctx, size_t nbytes) {
  void *ptr = pymalloc_alloc(mhd_state, ctx, nbytes);
  if (LIKELY(ptr != NULL)) {
    // Tag the ptr.
    mh_header* shdr = HEADER_PTR(ptr);
    shdr->pool_id = mhd_state->pool_id;
    shdr->mh_magic = mhd_state->magic;//MH_MAGIC_OBJ;
    return ptr;
  }
  /* We need to re-tag to let realloc know we had to reallocate inside raw.*/ 
  ptr = PyMem_RawMalloc(nbytes);
  if (ptr != NULL) {
    mh_header* shdr = HEADER_PTR(ptr);
    shdr->pool_id = mhd_state->pool_id;
    shdr->mh_magic = MH_NOT_MAGIC;
  }
  return ptr;
}

void *
_Extrn_Malloc(void *ctx, size_t nbytes)
{
    mh_state* mhd_state = mh_heaps_get_curr_heap();
    nbytes += HEADER_SZ;
    void* ptr = pymalloc_alloc(mhd_state, ctx, nbytes);
    if (LIKELY(ptr != NULL)) {
        mh_header* shdr = HEADER_PTR(ptr);
        shdr->pool_id = mhd_state->pool_id; 
        shdr->mh_magic = mhd_state->magic;//MH_MAGIC_OBJ; 
        return HEADER_TO_USER(shdr);
    }

    ptr = PyMem_RawMalloc(nbytes);
    if (ptr != NULL) {
        mhd_state->raw_allocated_blocks++;
        mh_header* shdr = HEADER_PTR(ptr);
        shdr->pool_id = mhd_state->pool_id;
        shdr->mh_magic = MH_NOT_MAGIC;
        return HEADER_TO_USER(shdr);
    }
    return NULL;
}

void *_Extrn_Calloc(void *ctx, size_t nelem, size_t elsize)
{
    assert(elsize == 0 || nelem <= (size_t)PY_SSIZE_T_MAX / elsize);
    mh_state* mhd_state = mh_heaps_get_curr_heap();
    size_t nbytes = nelem * elsize;
    nbytes += HEADER_SZ;
    void* ptr = pymalloc_alloc(mhd_state, ctx, nbytes);
    if (LIKELY(ptr != NULL)) {
        memset(ptr, 0, nbytes);
        mh_header *shdr = HEADER_PTR(ptr);
        shdr->pool_id = mhd_state->pool_id;
        shdr->mh_magic = mhd_state->magic;//MH_MAGIC_OBJ; 
        return HEADER_TO_USER(shdr);
    }

    //ptr = PyMem_RawCalloc(nelem, elsize);
    ptr = PyMem_RawMalloc(nbytes);
    if (ptr != NULL) {
        mhd_state->raw_allocated_blocks++;
        mh_header* shdr = HEADER_PTR(ptr);
        shdr->pool_id = mhd_state->pool_id;
        shdr->mh_magic = MH_NOT_MAGIC;
        return HEADER_TO_USER(shdr);
    }
    return ptr;
}

static void
insert_to_usedpool(mh_state* mhcurr, poolp pool)
{
    assert(pool->ref.count > 0);            /* else the pool is empty */

    uint size = pool->szidx;
    poolp next = mhcurr->usedpools[size + size];
    poolp prev = next->prevpool;

    /* insert pool before next:   prev <-> pool <-> next */
    pool->nextpool = next;
    pool->prevpool = prev;
    next->prevpool = pool;
    prev->nextpool = pool;
}

static void
insert_to_freepool(mh_state* mhcurr, poolp pool)
{
    poolp next = pool->nextpool;
    poolp prev = pool->prevpool;
    next->prevpool = prev;
    prev->nextpool = next;

    /* Link the pool to freepools.  This is a singly-linked
     * list, and pool->prevpool isn't used there.
     */
    struct arena_object *ao = &(mhcurr->arenas[pool->arenaindex]);
    pool->nextpool = ao->freepools;
    ao->freepools = pool;
    uint nf = ao->nfreepools;
    /* If this is the rightmost arena with this number of free pools,
     * mhcurr->nfp2lasta[nf] needs to change.  Caution:  if nf is 0, there
     * are no arenas in mhcurr->usable_arenas with that value.
     */
    struct arena_object* lastnf = mhcurr->nfp2lasta[nf];
    assert((nf == 0 && lastnf == NULL) ||
           (nf > 0 &&
            lastnf != NULL &&
            lastnf->nfreepools == nf &&
            (lastnf->nextarena == NULL ||
             nf < lastnf->nextarena->nfreepools)));
    if (lastnf == ao) {  /* it is the rightmost */
        struct arena_object* p = ao->prevarena;
        mhcurr->nfp2lasta[nf] = (p != NULL && p->nfreepools == nf) ? p : NULL;
    }
    ao->nfreepools = ++nf;

    /* All the rest is arena management.  We just freed
     * a pool, and there are 4 cases for arena mgmt:
     * 1. If all the pools are free, return the arena to
     *    the system free().  Except if this is the last
     *    arena in the list, keep it to avoid thrashing:
     *    keeping one wholly free arena in the list avoids
     *    pathological cases where a simple loop would
     *    otherwise provoke needing to allocate and free an
     *    arena on every iteration.  See bpo-37257.
     * 2. If this is the only free pool in the arena,
     *    add the arena back to the `mhcurr->usable_arenas` list.
     * 3. If the "next" arena has a smaller count of free
     *    pools, we have to "slide this arena right" to
     *    restore that mhcurr->usable_arenas is sorted in order of
     *    nfreepools.
     * 4. Else there's nothing more to do.
     */
    if (nf == ao->ntotalpools && ao->nextarena != NULL) {
        /* Case 1.  First unlink ao from mhcurr->usable_arenas.
         */
        assert(ao->prevarena == NULL ||
               ao->prevarena->address != 0);
        assert(ao ->nextarena == NULL ||
               ao->nextarena->address != 0);

        /* Fix the pointer in the prevarena, or the
         * mhcurr->usable_arenas pointer.
         */
        if (ao->prevarena == NULL) {
            mhcurr->usable_arenas = ao->nextarena;
            assert(mhcurr->usable_arenas == NULL ||
                   mhcurr->usable_arenas->address != 0);
        }
        else {
            assert(ao->prevarena->nextarena == ao);
            ao->prevarena->nextarena =
                ao->nextarena;
        }
        /* Fix the pointer in the nextarena. */
        if (ao->nextarena != NULL) {
            assert(ao->nextarena->prevarena == ao);
            ao->nextarena->prevarena =
                ao->prevarena;
        }
        /* Record that this arena_object slot is
         * available to be reused.
         */
        ao->nextarena = mhcurr->unused_arena_objects;
        mhcurr->unused_arena_objects = ao;

        /* Free the entire arena. */
        _PyObject_Arena.free(_PyObject_Arena.ctx,
                             (void *)ao->address, ARENA_SIZE);
        ao->address = 0;                        /* mark unassociated */
        --(mhcurr->narenas_currently_allocated);

        return;
    }

    if (nf == 1) {
        /* Case 2.  Put ao at the head of
         * mhcurr->usable_arenas.  Note that because
         * ao->nfreepools was 0 before, ao isn't
         * currently on the mhcurr->usable_arenas list.
         */
        ao->nextarena = mhcurr->usable_arenas;
        ao->prevarena = NULL;
        if (mhcurr->usable_arenas)
            mhcurr->usable_arenas->prevarena = ao;
        mhcurr->usable_arenas = ao;
        assert(mhcurr->usable_arenas->address != 0);
        if (mhcurr->nfp2lasta[1] == NULL) {
            mhcurr->nfp2lasta[1] = ao;
        }

        return;
    }

    /* If this arena is now out of order, we need to keep
     * the list sorted.  The list is kept sorted so that
     * the "most full" arenas are used first, which allows
     * the nearly empty arenas to be completely freed.  In
     * a few un-scientific tests, it seems like this
     * approach allowed a lot more memory to be freed.
     */
    /* If this is the only arena with nf, record that. */
    if (mhcurr->nfp2lasta[nf] == NULL) {
        mhcurr->nfp2lasta[nf] = ao;
    } /* else the rightmost with nf doesn't change */
    /* If this was the rightmost of the old size, it remains in place. */
    if (ao == lastnf) {
        /* Case 4.  Nothing to do. */
        return;
    }
    /* If ao were the only arena in the list, the last block would have
     * gotten us out.
     */
    assert(ao->nextarena != NULL);

    /* Case 3:  We have to move the arena towards the end of the list,
     * because it has more free pools than the arena to its right.  It needs
     * to move to follow lastnf.
     * First unlink ao from mhcurr->usable_arenas.
     */
    if (ao->prevarena != NULL) {
        /* ao isn't at the head of the list */
        assert(ao->prevarena->nextarena == ao);
        ao->prevarena->nextarena = ao->nextarena;
    }
    else {
        /* ao is at the head of the list */
        assert(mhcurr->usable_arenas == ao);
        mhcurr->usable_arenas = ao->nextarena;
    }
    ao->nextarena->prevarena = ao->prevarena;
    /* And insert after lastnf. */
    ao->prevarena = lastnf;
    ao->nextarena = lastnf->nextarena;
    if (ao->nextarena != NULL) {
        ao->nextarena->prevarena = ao;
    }
    lastnf->nextarena = ao;
    /* Verify that the swaps worked. */
    assert(ao->nextarena == NULL || nf <= ao->nextarena->nfreepools);
    assert(ao->prevarena == NULL || nf > ao->prevarena->nfreepools);
    assert(ao->nextarena == NULL || ao->nextarena->prevarena == ao);
    assert((mhcurr->usable_arenas == ao && ao->prevarena == NULL)
           || ao->prevarena->nextarena == ao);
}

/* Free a memory block allocated by pymalloc_alloc().
   Return 1 if it was freed.
   Return 0 if the block was not allocated by pymalloc_alloc(). */
static inline int
pymalloc_free(mh_state* mhcurr, void *ctx, void *p)
{
    assert(p != NULL);

#ifdef WITH_VALGRIND
    if (UNLIKELY(running_on_valgrind > 0)) {
        return 0;
    }
#endif

    poolp pool = POOL_ADDR(p);
    if (UNLIKELY(!address_in_range(mhcurr, p, pool))) {
        return 0;
    }
    /* We allocated this address. */

    /* Link p to the start of the pool's freeblock list.  Since
     * the pool had at least the p block outstanding, the pool
     * wasn't empty (so it's already in a usedpools[] list, or
     * was full and is in no list -- it's not in the freeblocks
     * list in any case).
     */
    assert(pool->ref.count > 0);            /* else it was empty */
    block *lastfree = pool->freeblock;
    *(block **)p = lastfree;
    pool->freeblock = (block *)p;
    pool->ref.count--;

    if (UNLIKELY(lastfree == NULL)) {
        /* Pool was full, so doesn't currently live in any list:
         * link it to the front of the appropriate usedpools[] list.
         * This mimics LRU pool usage for new allocations and
         * targets optimal filling when several pools contain
         * blocks of the same size class.
         */
        insert_to_usedpool(mhcurr, pool);
        return 1;
    }

    /* freeblock wasn't NULL, so the pool wasn't full,
     * and the pool is in a usedpools[] list.
     */
    if (LIKELY(pool->ref.count != 0)) {
        /* pool isn't empty:  leave it in usedpools */
        return 1;
    }

    /* Pool is now empty:  unlink from usedpools, and
     * link to the front of freepools.  This ensures that
     * previously freed pools will be allocated later
     * (being not referenced, they are perhaps paged out).
     */
    insert_to_freepool(mhcurr, pool);
    return 1;
}

static void
_Intrn_Free(mh_state* mhd_state, void *ctx, void *p) {
  if (p == NULL) {
    return;
  }
  if(UNLIKELY(!pymalloc_free(mhd_state, ctx, p))) {
    PyMem_RawFree(p);
    mhd_state->raw_allocated_blocks--;
  }
}

void
_Extrn_Free(void *ctx, void *p)
{
    /* PyObject_Free(NULL) has no effect */
    if (p == NULL) {
        return;
    }
    int64_t id = mh_get_id(p); 
    mh_header* shdr = USER_TO_HEADER(p); 
    assert(MH_IS_MAGIC(shdr->mh_magic) || shdr->mh_magic == MH_NOT_MAGIC); 
    p = VOID_PTR(shdr);
    mh_state* mhd_state = mh_heaps_get_heap(id, shdr->mh_magic);
    if (UNLIKELY(!pymalloc_free(mhd_state, ctx, p))) {
      /* pymalloc didn't allocate this address */
      PyMem_RawFree(p);
      mhd_state->raw_allocated_blocks--;
    }
}

/* pymalloc realloc.

   If nbytes==0, then as the Python docs promise, we do not treat this like
   free(p), and return a non-NULL result.

   Return 1 if pymalloc reallocated memory and wrote the new pointer into
   newptr_p.

   Return 0 if pymalloc didn't allocated p. */
static int
pymalloc_realloc(mh_state* mhcurr, void *ctx, void **newptr_p, void *p, size_t nbytes)
{
    void *bp;
    poolp pool;
    size_t size;

    assert(p != NULL);
    mh_header* shdr = HEADER_PTR(p);
    assert(MH_IS_MAGIC(shdr->mh_magic) || shdr->mh_magic == MH_NOT_MAGIC);
    assert(shdr->pool_id == mhcurr->pool_id);

#ifdef WITH_VALGRIND
    /* Treat running_on_valgrind == -1 the same as 0 */
    if (UNLIKELY(running_on_valgrind > 0)) {
        return 0;
    }
#endif

    pool = POOL_ADDR(p);
    if (!address_in_range(mhcurr, p, pool)) {
        /* pymalloc is not managing this block.

           If nbytes <= SMALL_REQUEST_THRESHOLD, it's tempting to try to take
           over this block.  However, if we do, we need to copy the valid data
           from the C-managed block to one of our blocks, and there's no
           portable way to know how much of the memory space starting at p is
           valid.

           As bug 1185883 pointed out the hard way, it's possible that the
           C-managed block is "at the end" of allocated VM space, so that a
           memory fault can occur if we try to copy nbytes bytes starting at p.
           Instead we punt: let C continue to manage this block. */
        return 0;
    }

    /* pymalloc is in charge of this block */
    size = INDEX2SIZE(pool->szidx);
    if (nbytes <= size) {
        /* The block is staying the same or shrinking.

           If it's shrinking, there's a tradeoff: it costs cycles to copy the
           block to a smaller size class, but it wastes memory not to copy it.

           The compromise here is to copy on shrink only if at least 25% of
           size can be shaved off. */
        if (4 * nbytes > 3 * size) {
            /* It's the same, or shrinking and new/old > 3/4. */
            *newptr_p = p;
            return 1;
        }
        size = nbytes;
    }

    bp = _Intrn_Malloc(mhcurr, ctx, nbytes);
    if (bp != NULL) {
        mh_header* shdr = HEADER_PTR(bp);
        uint32_t magic = shdr->mh_magic;
        memcpy(bp, p, size);
        /* Re-establish the correct header. */
        shdr->mh_magic = magic;
        _Intrn_Free(mhcurr, ctx, p);
    }
    *newptr_p = bp;
    return 1;
}

void *
_Extrn_Realloc(void *ctx, void *ptr, size_t nbytes)
{
    void *ptr2;

    if (ptr == NULL) {
      // This will set the header.
        return _Extrn_Malloc(ctx, nbytes);
    }
    // Already allocated, realloc in the same heap.
    int64_t id = mh_get_id(ptr); 
    uint32_t magic = USER_TO_HEADER(ptr)->mh_magic;
    mh_state* mhd_state = mh_heaps_get_heap(id, magic);
    ptr = VOID_PTR(USER_TO_HEADER(ptr));
    nbytes += HEADER_SZ;
    if (pymalloc_realloc(mhd_state, ctx, &ptr2, ptr, nbytes)) {
        mh_header* shdr = HEADER_PTR(ptr2);
        shdr->pool_id = mhd_state->pool_id;
        /* It might have gotten setup. */
        if (shdr->mh_magic != MH_NOT_MAGIC) {
          shdr->mh_magic = mhd_state->magic;//MH_MAGIC_OBJ;
        }
        return HEADER_TO_USER(shdr);
    }

    mh_header* shdr = HEADER_PTR(ptr);
    assert(shdr->mh_magic == MH_NOT_MAGIC);
    ptr2 = PyMem_RawRealloc(ptr, nbytes);
    shdr = HEADER_PTR(ptr2);
    shdr->pool_id = mhd_state->pool_id;
    shdr->mh_magic = MH_NOT_MAGIC;
    return HEADER_TO_USER(shdr);
}
