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
check Channel.bad state unreachable
*/
A[] not Channel4.bad
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and gen_func220_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and merge_func222_0.range_receiving_cid_var395_c_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and merge_func222_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sq_func221_0.range_receiving_cid_var387_in_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and sq_func221_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and sq_func221_1.range_receiving_cid_var387_in_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and sq_func221_1.select_pass_2_0)

