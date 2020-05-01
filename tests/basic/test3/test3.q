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
A[] not (deadlock and consumer_0.range_receiving_cid_var25_in_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and processor_0.range_receiving_cid_var22_in_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and processor_0.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and producer_0.sending_out_0)

