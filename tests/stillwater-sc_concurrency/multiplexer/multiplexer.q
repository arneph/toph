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
A[] not (deadlock and fanin1_func475_0.receiving_input1_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin1_func475_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin1_func476_0.receiving_input2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin1_func476_0.sending_c_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and fanin2_func477_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin2_func477_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin2_func477_0.sending_c_1)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and fanin3_func478_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin3_func478_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and fanin3_func478_0.sending_c_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and process_A_func479_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and process_Aprime_func481_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and process_B_func480_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and process_Bprime_func482_0.sending_c_0)

