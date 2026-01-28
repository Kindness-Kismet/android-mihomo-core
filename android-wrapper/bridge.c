#include "bridge.h"

void (*release_object_func)(void *obj);
void (*free_string_func)(char *data);
char *(*resolve_process_func)(void *tun_ctx, int protocol, const char *source, const char *target, int uid);
void (*protect_socket_func)(void *tun_ctx, int fd);
void (*result_func)(void *callback, const char *data);

void release_object(void *obj) {
    if (release_object_func == NULL) {
        return;
    }
    release_object_func(obj);
}

void free_string(char *data) {
    if (free_string_func == NULL) {
        return;
    }
    free_string_func(data);
}

char *resolve_process(void *tun_ctx, int protocol, const char *source, const char *target, int uid) {
    if (resolve_process_func == NULL) {
        return NULL;
    }
    return resolve_process_func(tun_ctx, protocol, source, target, uid);
}

void protect_socket(void *tun_ctx, int fd) {
    if (protect_socket_func == NULL) {
        return;
    }
    protect_socket_func(tun_ctx, fd);
}

void invoke_result(void *callback, const char *data) {
    if (result_func == NULL) {
        return;
    }
    result_func(callback, data);
}
