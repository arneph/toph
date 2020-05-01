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
A[] not (deadlock and filter_func199_0.receiving_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and filter_func199_0.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and filter_func199_0.receiving_c_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and filter_func199_0.sending_out_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and filter_func199_1.receiving_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and filter_func199_1.sending_out_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and filter_func199_1.receiving_c_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and filter_func199_1.sending_out_1)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and generate_func198_0.sending_c_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_c_0)

