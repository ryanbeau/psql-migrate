package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type Conn interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type Version struct {
	ID         int
	Major      uint
	Minor      uint
	Patch      uint
	StartedAt  time.Time
	FinishedAt time.Time
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Compare(o Version) int {
	compareSegment := func(v, o uint) int {
		if v < o {
			return -1
		}
		if v > o {
			return 1
		}
		return 0
	}

	if d := compareSegment(v.Major, o.Major); d != 0 {
		return d
	}
	if d := compareSegment(v.Minor, o.Minor); d != 0 {
		return d
	}
	if d := compareSegment(v.Patch, o.Patch); d != 0 {
		return d
	}
	return 0
}

func (v Version) GreaterThan(o Version) bool {
	return v.Compare(o) > 0
}

func (v Version) GreaterThanOrEq(o Version) bool {
	return v.Compare(o) >= 0
}

func (v Version) LessThanOrEq(o Version) bool {
	return v.Compare(o) <= 0
}

func Parse(semver string) (Version, error) {
	semver = strings.TrimSpace(semver)
	var version Version
	if split := strings.Split(semver, "."); len(split) == 3 {
		for i := 0; i < 3; i++ {
			v, err := strconv.Atoi(split[i])
			if err != nil && v <= 0 {
				break
			}

			if i == 0 {
				version.Major = uint(v)
			} else if i == 1 {
				version.Minor = uint(v)
			} else {
				version.Patch = uint(v)
				return version, nil
			}
		}
	}
	return version, fmt.Errorf("Invalid MAJOR.MINOR.PATCH semver: '%v'", semver)
}
