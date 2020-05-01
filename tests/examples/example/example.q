/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and producerA_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and producerA_0.sending_ch_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and producerA_0.sending_ch_2)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and producerB_0.sending_ch_0)

