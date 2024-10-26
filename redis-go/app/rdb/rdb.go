package rdb

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// TODO - this was from the replication stage which needed a hardcoded rdb consolidate this with the ReadRdb function below later
func ReadRdb() string {
	// hardcoded response for this stage
	//		see: https://github.com/codecrafters-io/redis-tester/blob/main/internal/assets/empty_rdb_hex.md
	return "UkVESVMwMDEx+glyZWRpcy12ZXIFNy4yLjD6CnJlZGlzLWJpdHPAQPoFY3RpbWXCbQi8ZfoIdXNlZC1tZW3CsMQQAPoIYW9mLWJhc2XAAP/wbjv+wP9aog=="
}

type RdbContents struct {
	Metadata  RedisMetadata
	Databases []RedisDatabase
}

type RedisMetadata struct {
	RedisVersion string
	Ctime        uint64
	UsedMem      uint64
	RedisBits    byte
}

type RedisDatabase struct {
	Keys     map[string]string
	Expiries map[string]uint64
}

const (
	RdbMetadataSeperator      = 0xFA
	RdbHashTableInfoSeperator = 0xFB
	RdbKeyExpiryMs            = 0xFC
	RdbKeyExpiryS             = 0xFD
	RdbDatabaseSeperator      = 0xFE
	RdbEofSeperator           = 0xFF
)

func ReadRdbFromFile(path string, filename string) (*RdbContents, error) {
	file, err := os.Open(filepath.Join(path, filename))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	// discard header (R E D I S 0 0 1 1)
	reader.Discard(9)

	result := RdbContents{
		Metadata:  RedisMetadata{},
		Databases: make([]RedisDatabase, 0),
	}

outerLoop:
	for {
		// read the seperator byte
		c, err := reader.ReadByte()
		fmt.Printf("[rdb] seperator: %x\n", c)
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		switch int(c) {
		case RdbMetadataSeperator:
			{
				fmt.Print("[rdb]reading metadata section\n")
				key, err := readStringValue(reader)
				if err != nil {
					return nil, err
				}

				switch key {
				case "redis-ver":
					{
						fmt.Print("\t[rdb]reading redis-ver\n")
						val, err := readStringValue(reader)
						if err != nil {
							return nil, fmt.Errorf("error reading redis-ver value: %w", err)
						}
						result.Metadata.RedisVersion = val
					}
				case "ctime":
					{
						fmt.Print("\t[rdb] reading ctime\n")
						reader.Discard(1) // TODO what is this seperator byte?? 194
						value, err := readUint32Value(reader)
						if err != nil {
							return nil, fmt.Errorf("error reading ctime value: %w", err)
						}

						result.Metadata.Ctime = uint64(value)
					}
				case "redis-bits":
					{
						fmt.Print("\t[rdb] reading redis-bits\n")
						reader.Discard(1) // TODO what is this seperator byte?? 192
						value, err := reader.ReadByte()
						if err != nil {
							return nil, fmt.Errorf("error reading redis-bits value: %w", err)
						}

						if value != 64 {
							return nil, fmt.Errorf("unexpected redis-bits got %d", value)
						}

						result.Metadata.RedisBits = value
					}
				case "used-mem":
					{
						fmt.Print("\t[rdb] reading used-mem\n")
						reader.Discard(1) // TODO what is this seperator byte?? 194
						value, err := readUint32Value(reader)
						if err != nil {
							return nil, fmt.Errorf("error reading used-mem value: %w", err)
						}

						result.Metadata.UsedMem = uint64(value)
					}
				case "aof-base":
					{
						fmt.Print("\t[rdb] reading aof-base\n")
						reader.Discard(2) // TODO unsure what this is - values 192, 0
					}
				default:
					return nil, fmt.Errorf("unexpected metadata key: %s", key)
				}
			}
		case RdbDatabaseSeperator:
			{
				fmt.Print("[rdb]reading database section\n")
				_, err := reader.ReadByte() // index of db
				if err != nil {
					return nil, fmt.Errorf("error reading database index %w", err)
				}

				b, err := reader.ReadByte() // hash table info
				if err != nil {
					return nil, fmt.Errorf("error reading hash table size %w", err)
				}

				if b != RdbHashTableInfoSeperator { // indicates hash table size information follows
					return nil, fmt.Errorf("expected hash table size info to follow but got %d", b)
				}

				ksize, err := reader.ReadByte() // key table size
				if err != nil {
					return nil, fmt.Errorf("error reading hash table size %w", err)
				}

				expsize, err := reader.ReadByte() // expiry table size
				if err != nil {
					return nil, fmt.Errorf("error reading hash table size %w", err)
				}

				database := &RedisDatabase{
					Keys:     make(map[string]string, ksize),
					Expiries: make(map[string]uint64, expsize),
				}

				result.Databases = append(result.Databases, *database)
				fmt.Print("\t[rdb]reading database\n")
				for i := 0; i < int(ksize); i++ {
					c, err := reader.Peek(1)
					if err != nil {
						return nil, fmt.Errorf("error peaking key expiry: %w", err)
					}

					hasExpiry := c[0] == RdbKeyExpiryMs || c[0] == RdbKeyExpiryS
					var expiry uint64 = 0
					if hasExpiry {
						reader.Discard(1) // discard the expiry indicator

						if c[0] == RdbKeyExpiryMs {
							fmt.Print("\t[rdb] has expiry in ms\n")
							expiry, err = readUint64Value(reader)
							if err != nil {
								return nil, fmt.Errorf("error reading expiry as uint64 %w", err)
							}
						} else {
							fmt.Print("\t[rdb] has expiry in s\n")
							e, err := readUint32Value(reader)
							if err != nil {
								return nil, fmt.Errorf("error reading expiry as uint32 %w", err)
							}

							expiry = uint64(e)
						}
					}

					t, err := reader.ReadByte() // type
					if err != nil {
						return nil, fmt.Errorf("error reading the key type")
					}
					fmt.Printf("\t[rdb] has type of %d\n", t)

					key, err := readStringValue(reader)
					if err != nil {
						return nil, fmt.Errorf("error reading key: %w", err)
					}

					if hasExpiry {
						database.Expiries[key] = expiry
					}

					fmt.Printf("\t[rdb] read key: %s\n", key)

					switch t {
					case 0: // string
						{
							b, _ := reader.Peek(2)
							fmt.Print(b)
							val, err := readStringValue(reader)
							if err != nil {
								return nil, fmt.Errorf("error reading val: %w", err)
							}

							fmt.Printf("\t[rdb] read value: %s\n", val)

							database.Keys[key] = val
						}
					default:
						{
							return nil, fmt.Errorf("unexpected key type: %d", t)
						}
					}
				}
			}
		case RdbEofSeperator:
			{
				fmt.Println("at end of file")
				chckbuf := make([]byte, 8)
				_, err := reader.Read(chckbuf)
				if err != nil {
					return nil, fmt.Errorf("error reading 8 byte checksum value: %w", err)
				}

				break outerLoop
			}
		default:
			{
				return nil, fmt.Errorf("unexpected seperator byte %v", c)
			}
		}
	}

	return &result, nil
}

func readStringValue(reader *bufio.Reader) (string, error) {
	len, err := reader.ReadByte()
	if err != nil {
		return "", fmt.Errorf("expected to read attr-val length but got error: %w", err)
	}

	valbuf := make([]byte, len)
	n, err := reader.Read(valbuf)
	if err != nil {
		return "", fmt.Errorf("error reading attr-val %w", err)
	}

	if n != int(len) {
		return "", fmt.Errorf("expected to read %d bytes when reading attr-val but got %d", len, n)
	}

	return string(valbuf), nil
}

func readUint64Value(reader *bufio.Reader) (uint64, error) {
	buf := make([]byte, 8)

	n, err := reader.Read(buf)

	if err != nil {
		return 0, fmt.Errorf("error reading int value: %w", err)
	}

	if n != 8 {
		return 0, fmt.Errorf("expected to read int of %d bytes but got %d", 8, n)
	}

	return binary.LittleEndian.Uint64(buf), nil
}

func readUint32Value(reader *bufio.Reader) (uint32, error) {
	buf := make([]byte, 4)

	n, err := reader.Read(buf)

	if err != nil {
		return 0, fmt.Errorf("error reading int value: %w", err)
	}

	if n != 4 {
		return 0, fmt.Errorf("expected to read int of %d bytes but got %d", 4, n)
	}

	return binary.LittleEndian.Uint32(buf), nil
}

func SerializeB64RdbToString(contents string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(contents)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.WriteString("$")
	builder.WriteString(strconv.Itoa(len(data)))
	builder.WriteString("\r\n")
	builder.Write(data)

	return builder.String(), nil
}

func DeserializeRdb(r io.Reader) (string, error) {
	reader := bufio.NewReader(r)
	reader.Read(make([]byte, 1)) // read $ prefix

	line := readLine(reader)

	count, err := strconv.Atoi(line)
	if err != nil {
		return "", err
	}

	data := make([]byte, count)

	_, err = reader.Read(data)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func readLine(r *bufio.Reader) string {
	line, _ := r.ReadString('\n')
	return line[:len(line)-2] // remove \r\n
}
