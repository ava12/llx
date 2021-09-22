/*
Incorrect struct. Should emit one error of each type:

 - incorrect indent
 - literal struct type
 - inconsistent field align
*/

typedef struct {
    int i[1];
  float f;
    struct {
        int a;
        float b;
    }   s;
} foo;
