/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check Channel.bad state unreachable
*/
A[] not Channel1.bad
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and ConcurrentSearchWithCutOff_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ConcurrentSearchWithCutOff_func265_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ConcurrentSearchWithCutOff_func266_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ConcurrentSearchWithCutOff_func267_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ConcurrentSearch_0.receiving_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ConcurrentSearch_func262_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ConcurrentSearch_func263_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ConcurrentSearch_func264_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and First_0.receiving_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and First_func270_0.sending_c_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and ReplicaSearch_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ReplicaSearch_func271_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ReplicaSearch_func272_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and ReplicaSearch_func273_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and time_after_func269_0.sending_ch_0)

