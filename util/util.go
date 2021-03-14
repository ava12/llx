package util

const IntSizeShift = 5 + (^uint(0) >> 32 & 1)
const IntSize = 1 << IntSizeShift
