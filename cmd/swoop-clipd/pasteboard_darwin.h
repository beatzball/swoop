#ifndef SWOOP_PASTEBOARD_H
#define SWOOP_PASTEBOARD_H

long swoop_pb_change_count(void);
int swoop_pb_concealed(void);
char *swoop_pb_text(void);
char *swoop_pb_types(void);
char *swoop_frontmost_app(void);

#endif
