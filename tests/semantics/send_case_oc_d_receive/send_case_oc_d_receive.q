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
A[] not (deadlock and main_func367_0.receiving_chA_0)

