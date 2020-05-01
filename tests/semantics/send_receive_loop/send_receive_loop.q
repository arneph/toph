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
A[] not (deadlock and main_func435_0.sending_chA_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main_func436_0.range_receiving_cid_var758_chA_0)

