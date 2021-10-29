package sdlang

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAst(t *testing.T) {
	code := `# a tag having only a name
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
	#     List rows = tag.getChild("matrix").getChildrenValues("content");`

	p := SaxParser{Input: code}
	ast, err := p.ParseIntoAst()
	assert.NoError(t, err)
	assert.Equal(t, 14, len(ast.Children))

	assert.Equal(t, "my_tag", ast.Children[0].Name)

	assert.Equal(t, "first_name", ast.Children[1].Name)
	assert.Equal(t, 1, len(ast.Children[1].Values))
	assert.True(t, ast.Children[1].Values[0].IsString())
	s, err := ast.Children[1].Values[0].String()
	assert.NoError(t, err)
	assert.Equal(t, "Akiko", s)

	assert.Equal(t, "last_name", ast.Children[2].Name)
	assert.Equal(t, 1, len(ast.Children[2].Values))
	assert.True(t, ast.Children[2].Values[0].IsString())
	s, err = ast.Children[2].Values[0].String()
	assert.NoError(t, err)
	assert.Equal(t, "Johnson", s)

	s, err = ast.Children[5].Attributes["first_name"].Value.String()
	assert.NoError(t, err)
	assert.Equal(t, "Akiko", s)
}
