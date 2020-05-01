/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel1.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel2.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Fork_0.sending_fork_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Fork_0.receiving_fork_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Fork_1.sending_fork_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Fork_1.receiving_fork_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Fork_2.sending_fork_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and Fork_2.receiving_fork_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and phil_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_0.sending_fork1_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_0.sending_fork1_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_0.sending_fork2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_0.sending_fork2_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_0.sending_fork2_2)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_0.sending_fork1_2)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and phil_1.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_1.sending_fork1_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_1.sending_fork1_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_1.sending_fork2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_1.sending_fork2_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_1.sending_fork2_2)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_1.sending_fork1_2)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and phil_2.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_2.sending_fork1_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_2.sending_fork1_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_2.sending_fork2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_2.sending_fork2_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_2.sending_fork2_2)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and phil_2.sending_fork1_2)

