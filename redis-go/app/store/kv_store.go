package store

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/rdb"
	"github.com/rs/zerolog"
)

type KvStore struct {
	logger   zerolog.Logger
	values   map[string]interface{}
	expiries map[string]uint64
	valqueue map[string]interface{}
	expqueue map[string]uint64
	mu       sync.RWMutex
}

type ValueOptions struct {
	Expiry uint64
}

func NewKvStore(logger zerolog.Logger) *KvStore {
	return &KvStore{
		values:   make(map[string]interface{}),
		expiries: make(map[string]uint64),
		valqueue: make(map[string]interface{}),
		expqueue: make(map[string]uint64),
		logger:   logger,
	}
}

func (k *KvStore) InitialiseFromRdbFile(filepath string, filename string) {
	k.logger.Info().Str("path", filepath).Str("filename", filename).Msg("loading rdb from file")

	file, err := rdb.ReadRdbFromFile(filepath, filename)

	if err != nil {
		if os.IsNotExist(err) {
			k.logger.Info().Msg("No rdb file found")
			return
		}

		k.logger.Fatal().Err(err).Msg("Error reading rdb file")
		os.Exit(1)
	}

	if len(file.Databases) == 0 {
		k.logger.Info().Msg("no databases found in rdb file")
		return
	}

	if len(file.Databases) > 1 {
		k.logger.Fatal().Int("databases", len(file.Databases)).Msg("currently only supports loading a single database from rdb file but got multiple")
	}

	db := file.Databases[0]
	k.logger.Info().Int("keycount", len(db.Keys)).Int("expirycount", len(db.Expiries)).Msg("loading db")

	k.mu.RLock()
	defer k.mu.RUnlock()

	for key, value := range db.Keys {
		k.values[key] = value
	}

	for key, value := range db.Expiries {
		k.expiries[key] = value
	}
}

func (k *KvStore) Get(key string) (interface{}, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	ms := currentMillis()
	expiry, exists := k.expiries[key]
	if exists && ms > expiry {
		delete(k.expiries, key)
		return "", false
	}

	val, exists := k.values[key]

	return val, exists
}

func (k *KvStore) List(pattern string) []string {
	k.mu.RLock()
	defer k.mu.RUnlock()

	var keys []string

	for key := range k.values {
		keys = append(keys, key) // TODO for this stage we are not filtering by pattern
	}

	return keys
}

func (k *KvStore) Set(key string, value interface{}, options ValueOptions) error {
	ms := currentMillis()
	k.mu.Lock()
	defer k.mu.Unlock()

	if options.Expiry != 0 {
		k.expiries[key] = ms + options.Expiry
	}

	k.values[key] = value

	return nil
}

type StreamKey = string
type StreamSeqKey = string
type StreamEntry = struct {
	Seqkey StreamSeqKey
	Values map[string]interface{}
}

type Stream = []StreamEntry

// {
// 	"stream_key": [
// 		{
// 			seqkey: "1526985054069-0"
// 			values: { "foo": "bar" }
// 		}
// 	]
// }

var (
	ErrInvalidStream   = errors.New("expected stream to be a stream")
	ErrStreamNotExists = errors.New("stream doesn't exist")
)

func (k *KvStore) SetStream(streamkey string, seqkey string, key string, value interface{}, options ValueOptions) (string, error) {
	ms := currentMillis()
	k.mu.Lock()
	defer k.mu.Unlock()

	stream, exists := k.values[streamkey]
	if !exists {
		stream = make(Stream, 0)
		k.values[streamkey] = stream
	}

	cast, ok := stream.(Stream)
	if !ok {
		return "", ErrInvalidStream
	}

	split := strings.Split(seqkey, "-")

	rtime := 0
	rseq := 0
	rseqwildcard := false

	if len(split) == 1 {
		rtime = int(ms)
		rseq = 0
	} else {
		rtime, _ = strconv.Atoi(split[0])
		r, rseqerr := strconv.Atoi(split[1])
		rseqwildcard = rseqerr != nil
		rseq = r

		if rseqwildcard && rtime == 0 {
			rseq = 1
		}
	}

	if rtime <= 0 && rseq <= 0 {
		return "", errors.New("ERR The ID specified in XADD must be greater than 0-0")
	}

	if len(cast) != 0 {
		last := cast[len(cast)-1]

		split = strings.Split(last.Seqkey, "-")
		ltime, _ := strconv.Atoi(split[0])
		lseq, _ := strconv.Atoi(split[1])

		if rseqwildcard && rtime == ltime {
			rseq = lseq + 1
		}

		if ltime > rtime || (ltime == rtime && lseq >= rseq) {
			return "", errors.New("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		}
	}

	seqkey = fmt.Sprintf("%d-%d", rtime, rseq)

	cast = append(cast, StreamEntry{Seqkey: seqkey, Values: map[string]interface{}{key: value}})
	k.values[streamkey] = cast

	return seqkey, nil
}

func (k *KvStore) GetStream(streamkey string, start string, end string) ([]StreamEntry, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	stream, exists := k.values[streamkey]
	if !exists {
		return make([]StreamEntry, 0), ErrStreamNotExists
	}

	cast, ok := stream.(Stream)
	if !ok {
		return make([]StreamEntry, 0), ErrInvalidStream
	}

	l := k.findl(start, cast)
	r := k.findr(end, cast)

	k.logger.Info().Int("start_i", l).Int("end_i", r).Msg("found stream search params")

	return cast[l:r], nil
}

func (k *KvStore) findl(start string, stream Stream) int {
	if start == "-" {
		return 0
	}

	starttime, startseq := SplitSeqKey(start)
	k.logger.Info().Int("start_t", starttime).Int("start_s", startseq).Msg("searching stream")

	l := 0
	for i, entry := range stream {
		ctime, cseq := SplitSeqKey(entry.Seqkey)

		c := compare(ctime, cseq, starttime, startseq)

		if c <= 0 {
			l = i
			break
		}
	}

	return l
}

func (k *KvStore) findr(end string, stream Stream) int {
	if end == "+" {
		return len(stream)
	}

	endtime, endseq := SplitSeqKey(end)
	k.logger.Info().Int("end_t", endtime).Int("end_s", endseq).Int("len", len(stream)).Msg("searching stream")

	r := len(stream) - 1
	for i := len(stream) - 1; i >= 0; i-- {
		entry := stream[i]
		ctime, cseq := SplitSeqKey(entry.Seqkey)

		c := compare(ctime, cseq, endtime, endseq)
		if c >= 0 {
			r = i + 1
			break
		}
	}

	return r
}

func (k *KvStore) XReadStream(streamkey string, start string) ([]StreamEntry, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	stream, exists := k.values[streamkey]
	if !exists {
		return make([]StreamEntry, 0), ErrStreamNotExists
	}

	cast, ok := stream.(Stream)
	if !ok {
		return make([]StreamEntry, 0), ErrInvalidStream
	}

	l := k.xreadfindl(start, cast)

	k.logger.Info().Int("start_i", l).Int("end_i", len(cast)).Msg("found stream search params")

	return cast[l:], nil
}

func (k *KvStore) xreadfindl(start string, stream Stream) int {
	starttime, startseq := SplitSeqKey(start)
	k.logger.Info().Int("start_t", starttime).Int("start_s", startseq).Msg("searching stream")

	l := 0
	for i, entry := range stream {
		ctime, cseq := SplitSeqKey(entry.Seqkey)

		c := compare(ctime, cseq, starttime, startseq)

		if c < 0 { // xread is exclusive - start needs to be bigger
			l = i
			break
		}
	}

	return l
}

func SplitSeqKey(key string) (int, int) {
	if key == "*" {
		return 0, 0
	}

	split := strings.Split(key, "-")
	time, _ := strconv.Atoi(split[0])

	seq := 0
	if len(split) > 1 {
		s, _ := strconv.Atoi(split[1])
		seq = s
	}

	return time, seq
}

func compare(ltime int, lseq int, rtime int, rseq int) int {
	if ltime == rtime && lseq == rseq {
		return 0 // same
	}

	if ltime > rtime || (ltime == rtime && lseq >= rseq) {
		return -1 // left bigger
	}

	return 1 // right bigger
}

func currentMillis() uint64 {
	return uint64(time.Now().UnixNano() / int64(time.Millisecond))
}
