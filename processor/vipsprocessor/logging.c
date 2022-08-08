#include "logging.h"

static void logging_handler(const gchar *log_domain,
                                   GLogLevelFlags log_level,
                                   const gchar *message, gpointer user_data) {
  goLoggingHandler((char *)log_domain, (int)log_level, (char *)message);
}

static void null_logging_handler(const gchar *log_domain,
                                 GLogLevelFlags log_level, const gchar *message,
                                 gpointer user_data) {}

void set_logging_handler(void) {
  g_log_set_default_handler(logging_handler, NULL);
}

void unset_logging_handler(void) {
  g_log_set_default_handler(null_logging_handler, NULL);
}
