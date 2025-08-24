package main

func orRunnerCh(runners []Runner) <-chan RunnerExitInfo {
	errChs := make([]<-chan RunnerExitInfo, 0, len(runners))
	for _, r := range runners {
		errChs = append(errChs, r.errCh)
	}
	return orRunnerExitInfoCh(errChs...)
}

func orRunnerExitInfoCh(chs ...<-chan RunnerExitInfo) <-chan RunnerExitInfo {
	switch len(chs) {
	case 0:
		return nil

	case 1:
		return chs[0]
	}

	orCh := make(chan RunnerExitInfo)

	switch len(chs) {
	case 2:

		go func() {
			defer close(orCh)
			select {
			case err := <-chs[0]:

				orCh <- err
			case err := <-chs[1]:

				orCh <- err
			}
		}()

	default:

		go func() {
			defer close(orCh)
			select {
			case err := <-chs[0]:

				orCh <- err
			case err := <-orRunnerExitInfoCh(chs[1:]...):

				orCh <- err
			}
		}()

	}
	return orCh
}
