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
A[] not (deadlock and rx_0.sending_reply_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and rx_0.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and tx_0.sending_snd_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and tx_0.select_pass_2_0)

