/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and consumer_0.range_receiving_cid_var637_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and consumer_1.range_receiving_cid_var637_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and consumer_2.range_receiving_cid_var637_ch_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and producer_0.sending_ch_0)

