%{

// +build ignore

package main

import (
    "bufio"
    "fmt"
    "math"
    "os"
)

%}

%union{
    value float64
}

%token  NUM

%left   '-' '+'
%left   '*' '/'
%left   NEG     /* negation--unary minus */
%right  '^'     /* exponentiation */

%type   <value> NUM, exp

%% /* The grammar follows.  */

input:    /* empty */
        | input line
;

line:     '\n'
        | exp '\n'  { fmt.Printf("\t%.10g\n", $1) }
;

exp:      NUM                { $$ = $1          }
        | exp '+' exp        { $$ = $1 + $3     }
        | exp '-' exp        { $$ = $1 - $3     }
        | exp '*' exp        { $$ = $1 * $3     }
        | exp '/' exp        { $$ = $1 / $3     }
        | '-' exp  %prec NEG { $$ = -$2         }
        | exp '^' exp        { $$ = math.Pow($1, $3) }
        | '(' exp ')'        { $$ = $2;         }
;
%%

func main() {
    os.Exit(yyParse(newLexer(bufio.NewReader(os.Stdin))))
}