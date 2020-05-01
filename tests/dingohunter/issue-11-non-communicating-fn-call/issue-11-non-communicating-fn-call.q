/*
check Channel.bad state unreachable
*/
A[] not Channel0.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_x_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main_func87_0.sending_x_0)

