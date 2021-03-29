#ifndef _MH_API_H
#define _MH_API_H

#include "mh_state.h"

typedef struct mh_header {
  uint32_t mh_magic;
  int64_t pool_id;
} mh_header;

typedef struct mh_heaps {
  mh_pkg** heaps;
  int64_t gen_id;
  size_t curr_size;
} mh_heaps;

typedef struct mh_stack_ids {
  int64_t* stack;
  size_t head;
  size_t size;
} mh_stack_ids;

#define MH_STACK_INIT_SZ 20
#define MH_STACK_GROWTH_FACTOR 2

#define BRK_INCREASE 0x2009000

#define MH_HEAPS_INIT_SZ 20 
#define MH_HEAPS_GROWTH_FACTOR 2

#define MH_MAGIC_OBJ (0xdeadbeef)
#define MH_MAGIC_FUNC  (0xdeadf04c)
#define MH_NOT_MAGIC (0xdeadbabe)

#define MH_IS_MAGIC(x) ((x) == MH_MAGIC_OBJ || (x) == MH_MAGIC_FUNC)

#define HEADER_SZ (sizeof(mh_header))

#define VOID_PTR(p) ((void *)p)
#define CHAR_PTR(p) ((char *)p)
#define HEADER_PTR(p) ((mh_header*)p)
#define USER_TO_HEADER(p) (HEADER_PTR((CHAR_PTR(p)-HEADER_SZ)))
#define HEADER_TO_USER(p) (VOID_PTR((CHAR_PTR(p)+HEADER_SZ)))

#define MH_SAVE_ALLOC(name) \
  int _##name = alloc_func; \
  alloc_func = 0;

#define MH_RESTORE_ALLOC(name) \
  alloc_func = _##name;


/* mh_heaps functions */

extern mh_heaps* all_heaps;
extern int alloc_func;
extern int SB_inside;

extern int mh_marker;

void mh_heaps_init();
mh_state* mh_heaps_get_curr_heap();
mh_state* mh_heaps_get_heap(int64_t id, uint32_t magic);

/* mh_stack functions */
void mh_stack_push(int64_t id);
int64_t mh_stack_peek();
int64_t mh_stack_pop();

/* Getting the id from a pointer. */
int64_t mh_get_id(void* ptr);

/* Checking if there is a danger. */
int mh_danger(void* ptr);

/* mh_pkg functions */
int64_t mh_new_pkg(const char* name);


/* Hooks for LitterBox callbacks. */
extern void (*register_id)(const char*, int);
extern void (*register_growth)(int, void*, size_t);
extern int (*check_readonly)(int);

#endif
