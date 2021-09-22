/*
Incorrect spaces and newlines. Should emit one error of each type:

 - missing space before {
 - excess space after {
 - int must be on a separate line
 - excess space before ;
 - more than one space
 - missing newline at file end
*/

typedef struct
{   int   i[1];
    float f;
    bar   s;
} foo ;

typedef struct {
    int   a;
    float b;
}  bar;