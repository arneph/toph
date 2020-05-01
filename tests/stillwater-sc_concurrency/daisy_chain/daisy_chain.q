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
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.sending_start_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main.receiving_sink_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and stage_0.receiving_left_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and stage_0.sending_right_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and stage_1.receiving_left_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and stage_1.sending_right_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and stage_2.receiving_left_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and stage_2.sending_right_0)

