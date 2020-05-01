/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_ch_0)

