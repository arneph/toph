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
A[] not (deadlock and main_func420_0.sending_chA_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main_func421_0.receiving_chA_0)

