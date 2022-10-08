package storage

type OrderByDate []Order

func (o OrderByDate) Len() int {
	return len(o)
}

func (o OrderByDate) Less(i, j int) bool {
	return o[i].UploadedAt.Before(o[j].UploadedAt)
}

func (o OrderByDate) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

type WithdrawalsByDate []Withdrawal

func (w WithdrawalsByDate) Len() int {
	return len(w)
}

func (w WithdrawalsByDate) Less(i, j int) bool {
	return w[i].ProcessedAt.Before(w[j].ProcessedAt)
}

func (w WithdrawalsByDate) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}
