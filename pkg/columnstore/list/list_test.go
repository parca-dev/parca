package list

/*
func Test_SingleList(t *testing.T) {
	l := &List{}

	for i := 0; i < 10; i++ {
		l.Prepend(&Node{
			part: i,
		})
	}

	j := 9
	l.Iterate(func(i int) bool {
		require.Equal(t, j, i)
		j--
		return true
	})
}

func Test_List_Concurrent(t *testing.T) {

	l := &List{}

	wg := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 10; i++ {
				l.Prepend(&Node{
					part: i,
				})
			}
		}(i)
	}
	wg.Wait()

	total := 0
	l.Iterate(func(i int) bool {
		total += i
		return true
	})

	require.Equal(t, 45*10, total)
}
*/
