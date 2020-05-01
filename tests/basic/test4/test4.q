/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.sending_ch_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.sending_ch_2)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_0.receiving_a_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_0.sending_b_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and test_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_0.sending_b_1)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and test_0.select_pass_2_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_0.receiving_b_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_0.sending_a_0)

