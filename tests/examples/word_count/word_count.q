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
check Channel.bad state unreachable
*/
A[] not Channel5.bad
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and countWords_0.range_receiving_cid_var657_filesChan_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and countWords_0.sending_errChan_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and countWords_0.sending_wordCountsChan_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and countWords_1.range_receiving_cid_var657_filesChan_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and countWords_1.sending_errChan_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and countWords_1.sending_wordCountsChan_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and findFilesInFolder_0.sending_errChan_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and findFilesInFolder_0.select_pass_2_0)
/*
check deadlock with blocked select statement unreachable
*/
A[] not (deadlock and main.select_pass_2_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main_func342_0.sending_waitChan_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main_func342_1.sending_waitChan_0)
/*
check deadlock with pending channel operation unreachable
*/
A[] not (deadlock and main_func343_0.receiving_waitChan_0)

