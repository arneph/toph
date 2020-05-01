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
A[] not (deadlock and forExample_0.receiving_chA_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and forExample_0.sending_chA_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and forExample_0.sending_chB_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ifExample_0.sending_chA_0)

