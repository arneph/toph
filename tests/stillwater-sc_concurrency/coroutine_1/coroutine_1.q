/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and rnd_time_loop_func459_0.sending_c_0)

