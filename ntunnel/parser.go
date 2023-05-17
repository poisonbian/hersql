package ntunnel

import (
	"fmt"
	"io"

	"hersql/pkg/bytesx"
	"github.com/dolthub/vitess/go/sqltypes"
	"github.com/dolthub/vitess/go/vt/proto/query"
	"go.uber.org/zap"
)

type Parser struct {
	body 	io.Reader
	logger  *zap.SugaredLogger
}

func NewParser(body io.Reader, logger *zap.SugaredLogger) *Parser {
	return &Parser{
		body: body,
		logger: logger,
	}
}

func (r *Parser) Parse() (result *sqltypes.Result, err error) {
	errno, err := r.parseHeader()
	if err != nil {
		r.logger.Errorf("parse header error!")
		return
	}

	if errno > 0 {
		var mysqlierr []byte
		mysqlierr, err = r.parseBlockValue()
		if err != nil {
			return
		}

		result = &sqltypes.Result{
			Info: string(mysqlierr),
		}
	} else {
		var errno, affectrows, insertid, numfields, numrows uint32
		errno, affectrows, insertid, numfields, numrows, err = r.parseResultSetHeader()
		if err != nil {
			return
		}

		if errno > 0 {
			var mysqlierr []byte
			mysqlierr, err = r.parseBlockValue()
			if err != nil {
				return
			}

			result = &sqltypes.Result{
				Info: string(mysqlierr),
			}
		} else {
			if numfields > 0 {
				var fields []*query.Field
				fields, err = r.parseFieldsHeader(numfields)
				if err != nil {
					return
				}

				var rows [][]sqltypes.Value
				rows, err = r.parseData(fields, numrows, numfields)
				if err != nil {
					return
				}

				result = &sqltypes.Result{
					Fields:       fields,
					RowsAffected: uint64(affectrows),
					InsertID:     uint64(insertid),
					Rows:         rows,
				}
			} else {
				var info []byte
				info, err = r.parseBlockValue()
				if err != nil {
					return
				}

				result = &sqltypes.Result{
					Info: string(info),
				}
			}
		}
	}

	return
}

func (r *Parser) parseHeader() (errno uint32, err error) {
	buf := make([]byte, 6)
	n, err := r.body.Read(buf)
	if err != nil {
		err = fmt.Errorf("Reader.parseHeader read head buf error: [%w]", err)
		return
	}
	if n != 6 {
		err = fmt.Errorf("Reader.parseHeader read head buf n:%d != 6", n)
		return
	}

	errno, err = r.parseUint32()
	if err != nil {
		err = fmt.Errorf("Reader.parseHeader read errno error: [%w]", err)
		return
	}

	n, err = r.body.Read(buf)
	if err != nil {
		err = fmt.Errorf("Reader.parseHeader read tail buf error: [%w]", err)
		return
	}
	if n != 6 {
		err = fmt.Errorf("Reader.parseHeader read tail buf n:%d != 6", n)
		return
	}

	return
}

func (r *Parser) parseResultSetHeader() (errno, affectrows, insertid, numfields, numrows uint32, err error) {
	errno, err = r.parseUint32()
	if err != nil {
		err = fmt.Errorf("Reader.parseResultSetHeader read errno error: [%w]", err)
		return
	}

	affectrows, err = r.parseUint32()
	if err != nil {
		err = fmt.Errorf("Reader.parseResultSetHeader read affectrows error: [%w]", err)
		return
	}

	insertid, err = r.parseUint32()
	if err != nil {
		err = fmt.Errorf("Reader.parseResultSetHeader read insertid error: [%w]", err)
		return
	}

	numfields, err = r.parseUint32()
	if err != nil {
		err = fmt.Errorf("Reader.parseResultSetHeader read numfields error: [%w]", err)
		return
	}

	numrows, err = r.parseUint32()
	if err != nil {
		err = fmt.Errorf("Reader.parseResultSetHeader read numrows error: [%w]", err)
		return
	}

	buf := make([]byte, 12)
	n, err := r.body.Read(buf)
	if err != nil {
		err = fmt.Errorf("Reader.parseResultSetHeader read buf error: [%w]", err)
		return
	}
	if n != 12 {
		err = fmt.Errorf("Reader.parseResultSetHeader read buf n:%d != 12", n)
		return
	}

	return
}

func (r *Parser) parseFieldsHeader(numfields uint32) (fields []*query.Field, err error) {
	fields = make([]*query.Field, numfields)

	var i uint32
	for ; i < numfields; i++ {
		var fieldName []byte
		fieldName, err = r.parseBlockValue()
		if err != nil {
			return
		}

		var fieldTable []byte
		fieldTable, err = r.parseBlockValue()
		if err != nil {
			return
		}

		var fieldType uint32
		fieldType, err = r.parseUint32()
		if err != nil {
			return
		}

		var fieldIntflag uint32
		fieldIntflag, err = r.parseUint32()
		if err != nil {
			return
		}

		var fieldLength uint32
		fieldLength, err = r.parseUint32()
		if err != nil {
			return
		}

		var queryFieldType query.Type
		queryFieldType, err = sqltypes.MySQLToType(int64(fieldType), int64(fieldIntflag))
		if err != nil {
			err = fmt.Errorf("Reader.parseFieldsHeader sqltypes.MySQLToType err:[%w]", err)
			return
		}

		fields[i] = &query.Field{
			Name:         string(fieldName),
			Table:        string(fieldTable),
			ColumnLength: uint32(fieldLength),
			Flags:        uint32(fieldIntflag),
			Type:         queryFieldType,
		}
	}

	return
}

func (r *Parser) parseData(fields []*query.Field, numrows, numfields uint32) (rows [][]sqltypes.Value, err error) {
	rows = make([][]sqltypes.Value, numrows)

	var i uint32
	for ; i < numrows; i++ {
		row := make([]sqltypes.Value, numfields)

		var j uint32
		for j = 0; j < numfields; j++ {
			// 先读一个字节判断是否是null
			var b byte
			b, err = r.parseByte()
			if err != nil {
				return
			}

			var val []byte
			var sqlval sqltypes.Value
			if b == '\xFF' {
				// 是null
				sqlval, err = sqltypes.NewValue(query.Type_NULL_TYPE, val)
			} else {
				val, err = r.parseBlockValueWithFirstByte(b)
				if err != nil {
					return
				}
				sqlval, err = sqltypes.NewValue(fields[j].Type, val)
			}

			if err != nil {
				err = fmt.Errorf("Reader.parseData sqltypes.NewValue error: [%w], name:[%s] type:[%d], val:[%v], b:[%v]", err, fields[j].Name, fields[j].Type, val, b)
				return
			}
			row[j] = sqlval
		}

		rows[i] = row
	}

	return
}

func (r *Parser) parseUint32() (value uint32, err error) {
	valueBys := make([]byte, 4)
	n, err := r.body.Read(valueBys)
	if err != nil {
		return
	}
	if n != 4 {
		err = fmt.Errorf("Reader.parseUint32 read n:[%d] != 4", n)
		return
	}

	value = bytesx.Bytes2Uint32(valueBys)

	return
}

func (r *Parser) parseBlockValue() (value []byte, err error) {
	lenBys := make([]byte, 1)
	n, err := r.body.Read(lenBys)
	if err != nil {
		err = fmt.Errorf("Reader.parseBlockValue read lenBys err:[%w]", err)
		return
	}
	if n != 1 {
		err = fmt.Errorf("Reader.parseBlockValue read lenBys n:[%d] != 1", n)
		return
	}

	var len uint32
	if lenBys[0] == '\xFE' {
		len, err = r.parseUint32()
		if err != nil {
			return
		}
	} else {
		len = uint32(lenBys[0])
	}

	value = make([]byte, len)
	n, err = r.body.Read(value)
	if err != nil {
		err = fmt.Errorf("Reader.parseBlockValue read value err:[%w]", err)
		return
	}
	if n != int(len) {
		err = fmt.Errorf("Reader.parseBlockValue read value n:[%d] != [%d]", n, len)
		return
	}

	return
}

func (r *Parser) parseBlockValueWithFirstByte(b byte) (value []byte, err error) {
	var len uint32
	if b == '\xFE' {
		len, err = r.parseUint32()
		if err != nil {
			return
		}
	} else {
		len = uint32(b)
	}

	value = make([]byte, len)
	n, err := r.body.Read(value)
	if err != nil {
		err = fmt.Errorf("Reader.parseBlockValueWithFirstByte read value err:[%w], value: %s", err, value)
		return
	}
	if n != int(len) {
		err = fmt.Errorf("Reader.parseBlockValueWithFirstByte read value n:[%d] != [%d], value: %s", n, len, value)
		return
	}

	return
}

func (r *Parser) parseByte() (b byte, err error) {
	bys := make([]byte, 1)
	n, err := r.body.Read(bys)
	if err != nil {
		return
	}
	if n != 1 {
		return
	}

	b = bys[0]

	return
}
