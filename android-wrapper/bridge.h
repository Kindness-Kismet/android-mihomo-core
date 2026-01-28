#pragma once

#include <stdlib.h>

// Host-injected callback function pointers (JNI).
extern void (*release_object_func)(void *obj);
extern void (*free_string_func)(char *data);
// Optional: resolve process name by socket 4-tuple (used by some hosts).
extern char *(*resolve_process_func)(void *tun_ctx, int protocol, const char *source, const char *target, int uid);
extern void (*protect_socket_func)(void *tun_ctx, int fd);
extern void (*result_func)(void *callback, const char *data);

void release_object(void *obj);
void free_string(char *data);
char *resolve_process(void *tun_ctx, int protocol, const char *source, const char *target, int uid);
void protect_socket(void *tun_ctx, int fd);
void invoke_result(void *callback, const char *data);
