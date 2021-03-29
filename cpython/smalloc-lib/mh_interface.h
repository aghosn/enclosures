#ifndef _MH_INTERFACE_H
#define _MH_INTERFACE_H


void *_Extrn_Malloc(void* ctx, size_t nbytes);
void *_Extrn_Calloc(void *ctx, size_t nelem, size_t elsize);
void *_Extrn_Realloc(void *ctx, void *ptr, size_t nbytes);
void _Extrn_Free(void *ctx, void *p);
#endif
