// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    model, err := UnmarshalModel(bytes)
//    bytes, err = model.Marshal()

package tmlanguage

import "encoding/json"

func UnmarshalModel(data []byte) (Model, error) {
	var r Model
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *Model) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

type Model struct {
	Patterns                                                                                    []Pattern          `json:"patterns"`
	// a dictionary (i.e. key/value pairs) of rules which can be included from other places in                     
	// the grammar. The key is the name of the rule and the value is the actual rule. Further                      
	// explanation (and example) follow with the description of the include rule key.                              
	Repository                                                                                  map[string]Pattern `json:"repository,omitempty"`
	// this is an array of file type extensions that the grammar should (by default) be used                       
	// with. This is referenced when TextMate does not know what grammar to use for a file the                     
	// user opens. If however the user selects a grammar from the language pop-up in the status                    
	// bar, TextMate will remember that choice.                                                                    
	FileTypes                                                                                   []string           `json:"fileTypes,omitempty"`
	FirstLineMatch                                                                              *string            `json:"firstLineMatch,omitempty"`
	// regular expressions that lines (in the document) are matched against. If a line matches                     
	// one of the patterns (but not both), it becomes a folding marker (see the foldings section                   
	// for more info).                                                                                             
	FoldingStartMarker                                                                          *string            `json:"foldingStartMarker,omitempty"`
	// regular expressions that lines (in the document) are matched against. If a line matches                     
	// one of the patterns (but not both), it becomes a folding marker (see the foldings section                   
	// for more info).                                                                                             
	FoldingStopMarker                                                                           *string            `json:"foldingStopMarker,omitempty"`
	Name                                                                                        *string            `json:"name,omitempty"`
	// this should be a unique name for the grammar, following the convention of being a                           
	// dot-separated name where each new (left-most) part specializes the name. Normally it                        
	// would be a two-part name where the first is either text or source and the second is the                     
	// name of the language or document type. But if you are specializing an existing type, you                    
	// probably want to derive the name from the type you are specializing. For example Markdown                   
	// is text.html.markdown and Ruby on Rails (rhtml files) is text.html.rails. The advantage                     
	// of deriving it from (in this case) text.html is that everything which works in the                          
	// text.html scope will also work in the text.html.«something» scope (but with a lower                         
	// precedence than something specifically targeting text.html.«something»).                                    
	ScopeName                                                                                   string             `json:"scopeName"`
	UUID                                                                                        *string            `json:"uuid,omitempty"`
}

type Pattern struct {
	ApplyEndPatternLast                                                                         *int64    `json:"applyEndPatternLast,omitempty"`
	// these keys allow matches which span several lines and must both be mutually exclusive              
	// with the match key. Each is a regular expression pattern. begin is the pattern that                
	// starts the block and end is the pattern which ends the block. Captures from the begin              
	// pattern can be referenced in the end pattern by using normal regular expression                    
	// back-references. This is often used with here-docs. A begin/end rule can have nested               
	// patterns using the patterns key.                                                                   
	Begin                                                                                       *string   `json:"begin,omitempty"`
	// allows you to assign attributes to the captures of the begin pattern. Using the captures           
	// key for a begin/end rule is short-hand for giving both beginCaptures and endCaptures with          
	// same values.                                                                                       
	BeginCaptures                                                                               *Captures `json:"beginCaptures,omitempty"`
	// allows you to assign attributes to the captures of the match pattern. Using the captures           
	// key for a begin/end rule is short-hand for giving both beginCaptures and endCaptures with          
	// same values.                                                                                       
	Captures                                                                                    *Captures `json:"captures,omitempty"`
	Comment                                                                                     *string   `json:"comment,omitempty"`
	// this key is similar to the name key but only assigns the name to the text between what is          
	// matched by the begin/end patterns.                                                                 
	ContentName                                                                                 *string   `json:"contentName,omitempty"`
	// set this property to 1 to disable the current pattern                                              
	Disabled                                                                                    *int64    `json:"disabled,omitempty"`
	// these keys allow matches which span several lines and must both be mutually exclusive              
	// with the match key. Each is a regular expression pattern. begin is the pattern that                
	// starts the block and end is the pattern which ends the block. Captures from the begin              
	// pattern can be referenced in the end pattern by using normal regular expression                    
	// back-references. This is often used with here-docs. A begin/end rule can have nested               
	// patterns using the patterns key.                                                                   
	End                                                                                         *string   `json:"end,omitempty"`
	// allows you to assign attributes to the captures of the end pattern. Using the captures             
	// key for a begin/end rule is short-hand for giving both beginCaptures and endCaptures with          
	// same values.                                                                                       
	EndCaptures                                                                                 *Captures `json:"endCaptures,omitempty"`
	// this allows you to reference a different language, recursively reference the grammar               
	// itself or a rule declared in this file's repository.                                               
	Include                                                                                     *string   `json:"include,omitempty"`
	// a regular expression which is used to identify the portion of text to which the name               
	// should be assigned. Example: '\b(true|false)\b'.                                                   
	Match                                                                                       *string   `json:"match,omitempty"`
	// the name which gets assigned to the portion matched. This is used for styling and                  
	// scope-specific settings and actions, which means it should generally be derived from one           
	// of the standard names.                                                                             
	Name                                                                                        *string   `json:"name,omitempty"`
	// applies to the region between the begin and end matches                                            
	Patterns                                                                                    []Pattern `json:"patterns,omitempty"`
	// these keys allow matches which span several lines and must both be mutually exclusive              
	// with the match key. Each is a regular expression pattern. begin is the pattern that                
	// starts the block and while continues it.                                                           
	While                                                                                       *string   `json:"while,omitempty"`
	// allows you to assign attributes to the captures of the while pattern. Using the captures           
	// key for a begin/while rule is short-hand for giving both beginCaptures and whileCaptures           
	// with same values.                                                                                  
	WhileCaptures                                                                               *Captures `json:"whileCaptures,omitempty"`
}

// allows you to assign attributes to the captures of the begin pattern. Using the captures
// key for a begin/end rule is short-hand for giving both beginCaptures and endCaptures with
// same values.
//
// allows you to assign attributes to the captures of the match pattern. Using the captures
// key for a begin/end rule is short-hand for giving both beginCaptures and endCaptures with
// same values.
//
// allows you to assign attributes to the captures of the end pattern. Using the captures
// key for a begin/end rule is short-hand for giving both beginCaptures and endCaptures with
// same values.
//
// allows you to assign attributes to the captures of the while pattern. Using the captures
// key for a begin/while rule is short-hand for giving both beginCaptures and whileCaptures
// with same values.
type Captures struct {
}
