/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel1.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and consume_0.receiving_msgs_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_done_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and produce_0.sending_msgs_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and produce_0.sending_done_0)

