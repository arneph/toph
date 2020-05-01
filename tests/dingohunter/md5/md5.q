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
A[] not (deadlock and MD5All_0.range_receiving_cid_var243_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and MD5All_0.receiving_errc_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sumFiles_func105_0.sending_errc_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and sumFiles_func105_func106_func107_0.select_pass_2_0)

