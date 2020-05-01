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
A[] not (deadlock and main.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.sending_ch_1)

