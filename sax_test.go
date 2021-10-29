package sdlang

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEmptyString(t *testing.T) {
	p := SaxParser{Input: ""}
	assert.NoError(t, p.Next())
	assert.True(t, p.IsEof())

	p = SaxParser{Input: " \t"}
	assert.NoError(t, p.Next())
	assert.True(t, p.IsEof())
}

func TestNewLine(t *testing.T) {
	p := SaxParser{Input: "\n \r\n\r"}
	assert.NoError(t, p.Next())
	assert.True(t, p.IsNewLine())
	assert.NoError(t, p.Next())
	assert.True(t, p.IsNewLine())
	assert.Error(t, p.Next())
}

func TestIdentifier(t *testing.T) {
	p := SaxParser{Input: "abc one:23="}
	assert.NoError(t, p.Next())
	assert.True(t, p.IsTagName())
	assert.Equal(t, "abc", p.Text())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsAttributeName())
	assert.Equal(t, "one", p.AdditionalText())
	assert.Equal(t, "23", p.Text())
}

func TestAttributeMustHaveEquals(t *testing.T) {
	p := SaxParser{Input: "t no:equals"}
	p.Next()
	assert.Error(t, p.Next())
}

func TestDoubleQuotedStringBasic(t *testing.T) {
	p := SaxParser{Input: "t \"ABC\" \"Unterminated"}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsString())
	assert.Equal(t, "ABC", p.Text())

	assert.Error(t, p.Next())
}

func TestDoubleQuotedStringInlineEscape(t *testing.T) {
	p := SaxParser{Input: `t "\r\n\t\"\\" "\¬"`}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsString())
	assert.Equal(t, "\r\n\t\"\\", p.Text())

	assert.Error(t, p.Next())
}

func TestDoubleQuotedStringLineEscape(t *testing.T) {
	p := SaxParser{Input: `t "john \
								doe"`}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsString())
	assert.Equal(t, "john doe", p.Text())
}

func TestContent(t *testing.T) {
	p := SaxParser{Input: `"This is content"`}

	assert.NoError(t, p.Next())
	assert.True(t, p.IsTagName())
	assert.Equal(t, "content", p.Text())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsString())
	assert.Equal(t, "This is content", p.Text())
}

func TestBacktickString(t *testing.T) {
	p := SaxParser{Input: "t `ab\nc` `unterminated"}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsString())
	assert.Equal(t, "ab\nc", p.Text())

	assert.Error(t, p.Next())

	p = SaxParser{Input: "t `\r\n`"}
	p.Next()
	assert.Error(t, p.Next())
}

func TestBinary(t *testing.T) {
	p := SaxParser{Input: "t [a\n b\nc] [unterminated"}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsBinary())
	assert.Equal(t, "abc", p.Text())

	assert.Error(t, p.Next())
}

func TestDate(t *testing.T) {
	p := SaxParser{Input: "t 1111/12/01"}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsDate())
	assert.Equal(t, time.Date(1111, time.Month(12), 01, 0, 0, 0, 0, time.UTC), p.Time())

	p = SaxParser{Input: "t 1111/1/11"}
	p.Next()
	assert.Error(t, p.Next())

	p = SaxParser{Input: "t 1111/11/1"}
	p.Next()
	assert.Error(t, p.Next())

	p = SaxParser{Input: "t 1aaa/bb/cc"}
	p.Next()
	assert.Error(t, p.Next())
}

func TestDateTime(t *testing.T) {
	p := SaxParser{Input: "t 1111/12/01 11:22:33.456"}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsDateTime())
	assert.Equal(t, time.Date(1111, time.Month(12), 01, 11, 22, 33, 456, time.UTC), p.Time())

	p = SaxParser{Input: "t 1111/11/11 22:bb:cc"}
	p.Next()
	assert.Error(t, p.Next())

	p = SaxParser{Input: "t 1111/11/11 11:22:3"}
	p.Next()
	assert.Error(t, p.Next())

	p = SaxParser{Input: "t 1111/11/11 11:22:33.45"}
	p.Next()
	assert.Error(t, p.Next())

	p = SaxParser{Input: "t 1111/11/11 11:22:33.45e"}
	p.Next()
	assert.Error(t, p.Next())
}

func TestNumber(t *testing.T) {
	p := SaxParser{Input: "t 123 123.456 -123 -123.456 123L 123.4F"}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsInteger())
	assert.Equal(t, "123", p.Text())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsDouble())
	assert.Equal(t, "123.456", p.Text())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsInteger())
	assert.Equal(t, "-123", p.Text())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsDouble())
	assert.Equal(t, "-123.456", p.Text())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsLong())
	assert.Equal(t, "123", p.Text())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsFloat())
	assert.Equal(t, "123.4", p.Text())

	p = SaxParser{Input: "t -1- 2.. 3b"}
	p.Next()

	assert.Error(t, p.Next())
	p.cursor = 5

	assert.Error(t, p.Next())
	p.cursor = 9

	assert.Error(t, p.Next())
}

func TestTimeSpan(t *testing.T) {
	p := SaxParser{Input: "t -55d:11:22:33.444 -00:02:30"}

	expected := ((time.Hour * 24 * 55) +
		(time.Hour * 11) +
		(time.Minute * 22) +
		(time.Second * 33) +
		(time.Millisecond * 444)) *
		-1
	p.Next()
	assert.NoError(t, p.Next())
	assert.True(t, p.IsTimeSpan())
	assert.Equal(t, expected, p.TimeSpan())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsTimeSpan())
}

func TestBoolean(t *testing.T) {
	p := SaxParser{Input: "t true on false off"}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsBool())
	assert.True(t, p.Bool())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsBool())
	assert.True(t, p.Bool())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsBool())
	assert.False(t, p.Bool())

	assert.NoError(t, p.Next())
	assert.True(t, p.IsBool())
	assert.False(t, p.Bool())
}

func TestNull(t *testing.T) {
	p := SaxParser{Input: "t null"}
	p.Next()

	assert.NoError(t, p.Next())
	assert.True(t, p.IsNull())
}

// Not testing the actual output (yet) because I'm lazy
// Also, keep last for obvious reasons >x3
func TestExamplesCanParse(t *testing.T) {
	// The formatter hates me.
	for _, example := range []string{
		`test "john \
		doe"
	`, `name "hello"
	line "he said \"hello there\""
	whitespace "item1\titem2\nitem3\titem4"
	continued "this is a long line \
		of text"`,
		`key [sdf789GSfsb2+3324sf2] name="my key"
		image [
			R3df789GSfsb2edfSFSDF
			uikuikk2349GSfsb2edfS
			vFSDFR3df789GSfsb2edf
		]
		upload from="ikayzo.org" data=[
			R3df789GSfsb2edfSFSDF
			uikuikk2349GSfsb2edfS
			vFSDFR3df789GSfsb2edf
		]
		`, `# create a tag called "date" with a date value of Dec 5, 2005
		date 2005/12/05
			 
		# a date time literal without a timezone
		here 2005/12/05 14:12:23.345`, `hours 03:00:00
		minutes 00:12:00
		seconds 00:00:42
		short_time 00:12:32.423
		long_time 30d:15:23:04.023
		before -00:02:30
		about_two_days_ago -2d:00:04:00 `, `ints 1 2 3
		doubles 5.0 3.1 6.4
		
		------------------
		
		lists {
			6 3 5 1
			"a" "r" "q"
			"bag" "of" "tricks"
		}`, `# a tag having only a name
		my_tag
		
		# three tags acting as name value pairs
		first_name "Akiko"
		last_name "Johnson"
		height 68
		
		# a tag with a value list
		person "Akiko" "Johnson" 68
		
		# a tag with attributes
		person first_name="Akiko" last_name="Johnson" height=68
		
		# a tag with values and attributes
		person "Akiko" "Johnson" height=60
		
		# a tag with attributes using namespaces
		person name:first-name="Akiko" name:last-name="Johnson"
		
		# a tag with values, attributes, namespaces, and children
		my_namespace:person "Akiko" "Johnson" dimensions:height=68 {
			son "Nouhiro" "Johnson"
			daughter "Sabrina" "Johnson" location="Italy" {
				hobbies "swimming" "surfing"
				languages "English" "Italian"
				smoker false
			}
		}   
		
		------------------------------------------------------------------
		// (notice the separator style comment above...)
		
		# a log entry
		#     note - this tag has two values (date_time and string) and an 
		#            attribute (error)
		entry 2005/11/23 10:14:23.253 "Something bad happened" error=true
		
		# a long line
		mylist "something" "another" true "shoe" 2002/12/13 "rock" \
			"morestuff" "sink" "penny" 12:15:23.425
		
		# a long string
		text "this is a long rambling line of text with a continuation \
		   and it keeps going and going..."
		   
		# anonymous tag examples
		
		files {
			"/folder1/file.txt"
			"/file2.txt"
		}
			
		# To retrieve the files as a list of strings
		#
		#     List files = tag.getChild("files").getChildrenValues("content");
		# 
		# We us the name "content" because the files tag has two children, each of 
		# which are anonymous tags (values with no name.)  These tags are assigned
		# the name "content"
			
		matrix {
			1 2 3
			4 5 6
		}
		
		# To retrieve the values from the matrix (as a list of lists)
		#
		#     List rows = tag.getChild("matrix").getChildrenValues("content");`} {
		p := SaxParser{Input: example}
		for !p.IsEof() && assert.NoError(t, p.Next()) {

		}
	}
}
