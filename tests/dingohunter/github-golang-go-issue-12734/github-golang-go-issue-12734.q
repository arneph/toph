/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_0.receiving_test_channel_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_a_0.sending_test_channel_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_a_1.sending_test_channel_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_a_2.sending_test_channel_0)

