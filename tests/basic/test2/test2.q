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
check Channel.bad state unreachable
*/
A[] not Channel3.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_func8_0.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_func8_1.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_func8_2.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_func8_3.sending_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_func9_0.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_func9_1.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_func9_2.receiving_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and test_func9_3.receiving_ch_0)

