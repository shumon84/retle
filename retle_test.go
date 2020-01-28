package retle

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestNewExpTimer(t *testing.T) {
	expectedExpTimer := &ExpTimer{
		interval:   time.Second,
		multiplier: 2.0,
	}
	actualExpTimer := NewExpTimer(time.Second, 2.0)

	if !reflect.DeepEqual(expectedExpTimer, actualExpTimer) {
		t.Fail()
	}
}

func TestDefaultExpTimer(t *testing.T) {
	expectedExpTimer := &ExpTimer{
		interval:   DefaultInitialInterval,
		multiplier: DefaultMultiplier,
	}
	actualExpTimer := DefaultExpTimer()

	if !reflect.DeepEqual(expectedExpTimer, actualExpTimer) {
		t.Fail()
	}
}

func TestExpTimer_NextDuration(t *testing.T) {
	expTimer := NewExpTimer(1, 2)
	actual := int64(1)

	// 10回連続NextDurationがexpectedな結果を返した場合は正しいことにする
	for i := 0; i < 10; i++ {
		expect := int64(expTimer.NextDuration())
		if actual != expect {
			t.Fail()
		}
		actual *= 2
	}
}

func TestExpTimer_Sleep(t *testing.T) {
	sleepTimer := DefaultExpTimer()
	durationTimer := DefaultExpTimer()

	startAt := time.Now()
	sleepTimer.Sleep()
	endAt := time.Now()
	actualElapsed := endAt.Sub(startAt)
	expectedElapsed := durationTimer.NextDuration()
	assertIsAvailableDuration(t, actualElapsed, expectedElapsed)
}

func TestExpTimer_Retry(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "リトライできる",
			run: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()

				expTimer := DefaultExpTimer()

				retryCount := 0 // 現在のリトライ回数
				retryLimit := 3 // リトライ回数の上限

				// リトライ回数の上限に達するまでリトライして、スリープ時間の合計を実測する
				overHead := time.Duration(0)
				startAt := time.Now()
				err := expTimer.Retry(ctx, func() (bool, error) {
					// スリープ以外の時間はオーバーヘッドになるので計測しておいて、あとで省く
					startAt := time.Now()
					defer func() {
						endAt := time.Now()
						overHead += endAt.Sub(startAt)
					}()

					// 上限までリトライさせる
					retryCount++
					if retryCount > retryLimit {
						return false, nil
					}
					return true, nil
				})
				// 終了時刻を取得して、スリープの経過時間を測る
				endAt := time.Now()
				if err != nil {
					t.Fatal(err)
				}
				// スリープ時間の合計 = 終了時刻 - 開始時刻 - スリープ以外の時間の合計
				actualElapsed := endAt.Sub(startAt) - overHead

				// 上限までリトライしたときの、期待されるスリープ時間の合計を計算
				expectedElapsed := time.Duration(0)
				durationTimer := DefaultExpTimer()
				for i := 0; i < retryLimit; i++ {
					expectedElapsed += durationTimer.NextDuration()
				}

				assertIsAvailableDuration(t, actualElapsed, expectedElapsed)
			},
		},
		{
			name: "キャンセルできる",
			run: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				time.AfterFunc(time.Second, cancel) // 1秒後にキャンセルする

				expTimer := DefaultExpTimer()
				err := expTimer.Retry(ctx, func() (bool, error) {
					return true, nil // かならずリトライさせる
				})
				if err != context.Canceled {
					t.Fatal(err)
				}
			},
		},
		{
			name: "タイムアウトできる",
			run: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second) // 1秒でタイムアウトするcontextを生成
				defer cancel()

				expTimer := DefaultExpTimer()
				err := expTimer.Retry(ctx, func() (bool, error) {
					return true, nil // かならずリトライさせる
				})
				if err != context.DeadlineExceeded {
					t.Fatal(err)
				}
			},
		},
		{
			name: "isRetryがfalseのとき正しいerrorを返す",
			run: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()

				expTimer := DefaultExpTimer()
				expectErr := errors.New("sample error")
				err := expTimer.Retry(ctx, func() (bool, error) {
					return false, expectErr // isRetryをfalseにし、errを返す
				})
				if err != expectErr {
					t.Fatal(err)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, testCase.run)
	}
}

func TestRetry(t *testing.T) {
	testCases := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "リトライできる",
			run: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()

				retryCount := 0 // 現在のリトライ回数
				retryLimit := 3 // リトライ回数の上限

				// リトライ回数の上限に達するまでリトライして、スリープ時間の合計を実測する
				overHead := time.Duration(0)
				startAt := time.Now()
				err := Retry(ctx, func() (bool, error) {
					// スリープ以外の時間はオーバーヘッドになるので計測しておいて、あとで省く
					startAt := time.Now()
					defer func() {
						endAt := time.Now()
						overHead += endAt.Sub(startAt)
					}()

					// 上限までリトライさせる
					retryCount++
					if retryCount > retryLimit {
						return false, nil
					}
					return true, nil
				})
				// 終了時刻を取得して、スリープの経過時間を測る
				endAt := time.Now()
				if err != nil {
					t.Fatal(err)
				}
				// スリープ時間の合計 = 終了時刻 - 開始時刻 - スリープ以外の時間の合計
				actualElapsed := endAt.Sub(startAt) - overHead

				// 上限までリトライしたときの、期待されるスリープ時間の合計を計算
				expectedElapsed := time.Duration(0)
				durationTimer := DefaultExpTimer()
				for i := 0; i < retryLimit; i++ {
					expectedElapsed += durationTimer.NextDuration()
				}

				assertIsAvailableDuration(t, actualElapsed, expectedElapsed)
			},
		},
		{
			name: "キャンセルできる",
			run: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				time.AfterFunc(time.Second, cancel) // 1秒後にキャンセルする

				err := Retry(ctx, func() (bool, error) {
					return true, nil // かならずリトライさせる
				})
				if err != context.Canceled {
					t.Fatal(err)
				}
			},
		},
		{
			name: "タイムアウトできる",
			run: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second) // 1秒でタイムアウトするcontextを生成
				defer cancel()

				err := Retry(ctx, func() (bool, error) {
					return true, nil // かならずリトライさせる
				})
				if err != context.DeadlineExceeded {
					t.Fatal(err)
				}
			},
		},
		{
			name: "isRetryがfalseのとき正しいerrorを返す",
			run: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
				defer cancel()

				expectErr := errors.New("sample error")
				err := Retry(ctx, func() (bool, error) {
					return false, expectErr // isRetryをfalseにし、errを返す
				})
				if err != expectErr {
					t.Fatal(err)
				}
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, testCase.run)
	}
}

func assertIsAvailableDuration(t *testing.T, actualElapsed, expectedElapsed time.Duration) {
	// 実測したスリープ時間と、期待されるスリープ時間の差が20msを超えている場合落とす
	if actualElapsed < expectedElapsed || actualElapsed-expectedElapsed > 20*time.Millisecond {
		t.Fatal("実測したスリープ時間と、期待されるスリープ時間の差が20msを超えています")
	} else {
		t.Log("(actual expected actual-expected) : (", actualElapsed.Round(time.Microsecond), expectedElapsed.Round(time.Microsecond), (actualElapsed - expectedElapsed).Round(time.Microsecond), ")")
	}
}
