package repository

import "time"

type DailyCount struct {
        Period time.Time
        Count  int64
}
