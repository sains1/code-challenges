package rdb

import "testing"

/* File format:

header:
	52 45 44 49 53 30 30 31 31  // Magic string + version number (ASCII): "REDIS0011".

metadata:
	FA                             // Indicates the start of a metadata subsection.
	09 72 65 64 69 73 2D 76 65 72  // The name of the metadata attribute (string encoded): "redis-ver".
	06 36 2E 30 2E 31 36           // The value of the metadata attribute (string encoded): "6.0.16".

database:
	FE                       // Indicates the start of a database subsection.
	00                       // The index of the database (size encoded). Here, the index is 0.
	FB                       // Indicates that hash table size information follows.
	03                       // The size of the hash table that stores the keys and values (size encoded). Here, the total key-value hash table size is 3.
	02                       // The size of the hash table that stores the expires of the keys (size encoded). Here, the number of keys with an expiry is 2.

	00						 // The 1-byte flag that specifies the value’s type and encoding. Here, the flag is 0, which means "string."
	06 66 6F 6F 62 61 72     // The name of the key (string encoded). Here, it's "foobar".
	06 62 61 7A 71 75 78     // The value (string encoded). Here, it's "bazqux".

	FC						 // Indicates that this key ("foo") has an expire, and that the expire timestamp is expressed in milliseconds
	15 72 E7 07 8F 01 00 00  // The expire timestamp, expressed in Unix time stored as an 8-byte unsigned long, in little-endian (read right-to-left). Here, the expire timestamp is 1713824559637
	00                       // Value type is string.
	03 66 6F 6F              // Key name is "foo".
	03 62 61 72              // Value is "bar".

	FD                       // Indicates that this key ("baz") has an expire, and that the expire timestamp is expressed in seconds.
	52 ED 2A 66              // The expire timestamp, expressed in Unix time, stored as a 4-byte unsigned integer, in little-endian (read right-to-left). Here, the expire timestamp is 1714089298.
	00                       // Value type is string.
	03 62 61 7A              // Key name is "baz".
	03 71 75 78              // Value is "qux".

eof:
	FF                       // Indicates that the file is ending, and that the checksum follows.
	89 3b b7 4e f8 0f 77 19  // An 8-byte CRC64 checksum of the entire file.
*/

func TestReadEmptyRdbFromFile(t *testing.T) {
	// arrange

	// act
	result, err := ReadRdbFromFile("../../.dumps/", "empty.rdb")

	// assert
	if err != nil {
		t.Error(err)
	}

	if len(result.Databases) > 0 {
		t.Error("expected rdb to contain no databases")
	}
}

func TestReadRdbWithValuesFromFile(t *testing.T) {
	// arrange

	// act
	result, err := ReadRdbFromFile("../../.dumps/", "with_key.rdb")

	// assert
	if err != nil {
		t.Error(err)
	}

	if len(result.Databases) != 1 {
		t.Errorf("expected rdb to contain exactly one database but was %d", len(result.Databases))
	}

	expectedKey := "mykey"
	expectedValue := "myval"

	database := result.Databases[0]

	val, exists := database.Keys[expectedKey]
	if !exists {
		t.Errorf("%s didn't exist in database values", expectedKey)
	}

	if val != expectedValue {
		t.Errorf("expected key to have value %s but got %s", expectedValue, val)
	}

	_, exists = database.Expiries[expectedKey]
	if exists {
		t.Error("expected key not to exist in expiries but got value")
	}

}

func TestReadRdbWithValuesAndExpiryInMillisFromFile(t *testing.T) {
	// arrange

	// act
	result, err := ReadRdbFromFile("../../.dumps/", "with_key_expiry_ms.rdb")

	// assert
	if err != nil {
		t.Error(err)
	}

	if len(result.Databases) != 1 {
		t.Errorf("expected rdb to contain exactly one database but was %d", len(result.Databases))
	}

	expectedKey := "mykey"
	expectedValue := "foo"
	var expectedExpiry uint64 = 1729939775013

	database := result.Databases[0]

	val, exists := database.Keys[expectedKey]
	if !exists {
		t.Errorf("%s didn't exist in database values", expectedKey)
	}

	if val != expectedValue {
		t.Errorf("expected key to have value %s but got %s", expectedValue, val)
	}

	expiry, exists := database.Expiries[expectedKey]
	if !exists {
		t.Errorf("%s didn't exist in database expiries", expectedKey)
	}

	if expiry != expectedExpiry {
		t.Errorf("expected key to have expiry %d but got %d", expectedExpiry, expiry)
	}
}
