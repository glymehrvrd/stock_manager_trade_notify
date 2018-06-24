package main

import "testing"

func TestGetManagerTradeInfo(t *testing.T) {
	infos, _ := GetManagerTradeInfo("002502")
	for _, info := range infos {
		t.Logf("%+v", info)
	}
}

func TestComma(t *testing.T) {
	t.Logf("111->%s",comma(111))
	t.Logf("1111->%s",comma(1111))
	t.Logf("111111111->%s",comma(111111111))
	t.Logf("-111->%s",comma(-111))
	t.Logf("0->%s",comma(0))
	t.Logf("-1111111->%s",comma(-1111111))
}

