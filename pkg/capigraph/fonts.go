package capigraph

type FontWeight int

const (
	FontWeightBold FontWeight = iota
	FontWeightNormal
)

func FontWeightToString(w FontWeight) string {
	switch w {
	case FontWeightBold:
		return "bold"
	case FontWeightNormal:
		return "normal"
	default:
		return "normal"
	}
}

type FontTypeface int

const (
	FontTypefaceArial FontTypeface = iota
	FontTypefaceCourier
	FontTypefaceVerdana
)

func FontTypefaceToString(tf FontTypeface) string {
	switch tf {
	case FontTypefaceArial:
		return "arial"
	case FontTypefaceCourier:
		return "courier"
	case FontTypefaceVerdana:
		return "verdana"
	default:
		return "courier"
	}
}

type FontOptions struct {
	Typeface     FontTypeface
	Weight       FontWeight
	SizeInPixels float64
	Interval     float64
}

// Generated by https://github.com/Evgenus/js-server-text-width
// 0x0: Basic Latin, Latin-1 Supplement, Latin Extended-A, Latin Extended-B
// 0x370: Greek and Coptic, Cyrillic
// Normal (400) and bold (700), Arial, Courier, Verdana font size 100px
var sizeMap map[FontTypeface]map[FontWeight]map[int][]int = map[FontTypeface]map[FontWeight]map[int][]int{
	FontTypefaceArial: {
		FontWeightNormal: {
			// "Arial|100px|400|0"
			0: []int{75, 75, 75, 75, 75, 75, 75, 75, 75, 28, 28, 28, 28, 28, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 28, 28, 35, 56, 56, 89, 67, 19, 33, 33, 39, 58, 28, 33, 28, 28, 56, 56, 56, 56, 56, 56, 56, 56, 56, 56, 28, 28, 58, 58, 58, 56, 102, 67, 67, 72, 72, 67, 61, 78, 72, 28, 50, 67, 56, 83, 72, 78, 67, 78, 72, 67, 61, 72, 67, 94, 67, 67, 61, 28, 28, 28, 47, 56, 33, 56, 56, 50, 56, 56, 28, 56, 56, 22, 22, 50, 22, 83, 56, 56, 56, 56, 33, 50, 28, 56, 50, 72, 50, 50, 50, 33, 26, 33, 58, 50, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 28, 33, 56, 56, 56, 56, 26, 56, 33, 74, 37, 56, 58, 0, 74, 55, 40, 55, 33, 33, 33, 58, 54, 33, 33, 33, 37, 56, 83, 83, 83, 61, 67, 67, 67, 67, 67, 67, 100, 72, 67, 67, 67, 67, 28, 28, 28, 28, 72, 72, 78, 78, 78, 78, 78, 58, 78, 72, 72, 72, 72, 67, 67, 61, 56, 56, 56, 56, 56, 56, 89, 50, 56, 56, 56, 56, 28, 28, 28, 28, 56, 56, 56, 56, 56, 56, 56, 55, 61, 56, 56, 56, 56, 50, 56, 50, 67, 56, 67, 56, 67, 56, 72, 50, 72, 50, 72, 50, 72, 50, 72, 61, 72, 56, 67, 56, 67, 56, 67, 56, 67, 56, 67, 56, 78, 56, 78, 56, 78, 56, 78, 56, 72, 56, 72, 56, 28, 28, 28, 28, 28, 28, 28, 22, 28, 28, 73, 44, 50, 22, 67, 50, 50, 56, 22, 56, 22, 56, 29, 56, 33, 56, 22, 72, 56, 72, 56, 72, 56, 60, 72, 56, 78, 56, 78, 56, 78, 56, 100, 94, 72, 33, 72, 33, 72, 33, 67, 50, 67, 50, 67, 50, 67, 50, 61, 28, 61, 38, 61, 28, 72, 56, 72, 56, 72, 56, 72, 56, 72, 56, 72, 56, 94, 72, 67, 50, 67, 61, 50, 61, 50, 61, 50, 22, 56, 76, 66, 56, 66, 56, 72, 72, 50, 72, 81, 66, 56, 56, 67, 75, 60, 61, 56, 78, 62, 88, 22, 28, 67, 50, 22, 50, 89, 72, 56, 78, 86, 66, 87, 67, 75, 56, 67, 67, 50, 62, 38, 28, 61, 28, 61, 85, 67, 75, 72, 77, 50, 61, 50, 61, 61, 54, 54, 56, 56, 46, 49, 56, 26, 41, 58, 28, 133, 122, 105, 106, 83, 45, 122, 94, 77, 67, 56, 28, 22, 78, 56, 72, 56, 72, 56, 72, 56, 72, 56, 72, 56, 56, 67, 56, 67, 56, 100, 89, 78, 56, 78, 56, 67, 50, 78, 56, 78, 56, 61, 54, 22, 133, 122, 105, 78, 56, 103, 62, 72, 56, 67, 56, 100, 89, 78, 61, 67, 56, 67, 56, 67, 56, 67, 56, 28, 28, 28, 28, 78, 56, 78, 56, 72, 33, 72, 33, 72, 56, 72, 56, 67, 50, 61, 28, 54, 44, 72, 56, 71, 68, 60, 57, 61, 50, 67, 56, 67, 56, 78, 56, 78, 56, 78, 56, 78, 56, 67, 50, 35, 68, 37, 22, 89, 89, 67, 72, 50, 56, 61, 50, 50, 58, 45, 67, 72, 67, 67, 56, 50, 22, 74, 56, 72, 33, 67, 50}, // "Arial|100px|400|0"
			// "Arial|100px|400|370"
			0x370: []int{58, 48, 61, 46, 33, 33, 72, 58, 75, 75, 33, 50, 50, 50, 28, 50, 75, 75, 75, 75, 33, 33, 67, 28, 78, 84, 38, 75, 77, 75, 86, 75, 22, 67, 67, 55, 67, 67, 61, 72, 78, 28, 67, 67, 83, 72, 65, 78, 72, 67, 75, 62, 61, 67, 80, 67, 84, 75, 28, 67, 58, 45, 56, 22, 55, 58, 58, 50, 56, 45, 44, 56, 56, 22, 50, 50, 58, 50, 45, 56, 69, 57, 48, 62, 40, 55, 65, 52, 71, 78, 22, 55, 56, 55, 78, 67, 58, 55, 77, 96, 77, 56, 78, 60, 78, 56, 72, 50, 61, 40, 62, 53, 76, 58, 89, 83, 67, 56, 67, 50, 67, 67, 61, 60, 74, 55, 46, 41, 60, 57, 50, 22, 78, 44, 44, 67, 56, 72, 83, 69, 57, 72, 72, 72, 67, 67, 86, 54, 72, 67, 28, 28, 50, 106, 101, 85, 58, 72, 64, 72, 67, 66, 67, 54, 68, 67, 92, 60, 72, 72, 58, 66, 83, 72, 78, 72, 67, 72, 61, 64, 76, 67, 74, 67, 92, 94, 79, 89, 66, 72, 101, 72, 56, 57, 53, 36, 58, 56, 67, 46, 56, 56, 44, 58, 69, 55, 56, 54, 56, 50, 46, 50, 82, 50, 57, 52, 80, 82, 63, 72, 52, 51, 75, 54, 56, 56, 56, 36, 51, 50, 22, 28, 22, 91, 81, 56, 44, 56, 50, 55, 134, 62, 78, 61, 95, 71, 67, 50, 90, 70, 83, 69, 105, 87, 60, 46, 80, 69, 78, 56, 80, 63, 80, 63, 107, 90, 83, 61, 119, 85, 134, 62, 72, 50, 50, 0, 0, 0, 0, 0, 0, 0, 72, 56, 66, 52, 67, 56, 49, 41, 54, 36, 67, 55, 92, 67, 60, 46, 58, 44, 58, 44, 58, 44, 74, 54, 72, 55, 88, 65, 114, 87, 75, 52, 72, 50, 61, 46, 56, 50, 56, 50, 67, 50, 93, 69, 67, 52, 67, 52, 67, 56, 86, 67, 86, 67, 28, 92, 67, 67, 55, 66, 58, 72, 55, 72, 55, 67, 52, 83, 69, 22, 67, 56, 67, 56, 100, 89, 67, 56, 75, 56, 75, 56, 92, 67, 60, 46, 60, 54, 72, 56, 72, 56, 78, 56, 78, 56, 78, 56, 72, 51, 64, 50, 64, 50, 64, 50, 67, 52, 54, 36, 89, 72, 54, 36, 67, 50, 67, 50}, // "Arial|100px|400|370"
		},
		FontWeightBold: {
			// "Arial|100px|700|0"
			0: []int{75, 75, 75, 75, 75, 75, 75, 75, 75, 28, 28, 28, 28, 28, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 28, 33, 47, 56, 56, 89, 72, 24, 33, 33, 39, 58, 28, 33, 28, 28, 56, 56, 56, 56, 56, 56, 56, 56, 56, 56, 33, 33, 58, 58, 58, 61, 98, 72, 72, 72, 72, 67, 61, 78, 72, 28, 56, 72, 61, 83, 72, 78, 67, 78, 72, 67, 61, 72, 67, 94, 67, 67, 61, 33, 28, 33, 58, 56, 33, 56, 61, 56, 61, 56, 33, 61, 61, 28, 28, 56, 28, 89, 61, 61, 61, 61, 39, 56, 33, 61, 56, 78, 56, 56, 50, 39, 28, 39, 58, 50, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 75, 28, 33, 56, 56, 56, 56, 28, 56, 33, 74, 37, 56, 58, 0, 74, 55, 40, 55, 33, 33, 33, 58, 56, 33, 33, 33, 37, 56, 83, 83, 83, 61, 72, 72, 72, 72, 72, 72, 100, 72, 67, 67, 67, 67, 28, 28, 28, 28, 72, 72, 78, 78, 78, 78, 78, 58, 78, 72, 72, 72, 72, 67, 67, 61, 56, 56, 56, 56, 56, 56, 89, 56, 56, 56, 56, 56, 28, 28, 28, 28, 61, 61, 61, 61, 61, 61, 61, 55, 61, 61, 61, 61, 61, 56, 61, 56, 72, 56, 72, 56, 72, 56, 72, 56, 72, 56, 72, 56, 72, 56, 72, 72, 72, 61, 67, 56, 67, 56, 67, 56, 67, 56, 67, 56, 78, 61, 78, 61, 78, 61, 78, 61, 72, 61, 72, 61, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 78, 56, 56, 28, 72, 56, 56, 61, 28, 61, 28, 61, 39, 61, 48, 61, 28, 72, 61, 72, 61, 72, 61, 71, 72, 61, 78, 61, 78, 61, 78, 61, 100, 94, 72, 39, 72, 39, 72, 39, 67, 56, 67, 56, 67, 56, 67, 56, 61, 33, 61, 48, 61, 33, 72, 61, 72, 61, 72, 61, 72, 61, 72, 61, 72, 61, 94, 78, 67, 56, 67, 61, 50, 61, 50, 61, 50, 28, 61, 84, 72, 61, 72, 61, 72, 72, 56, 72, 84, 72, 61, 61, 67, 73, 63, 61, 56, 78, 64, 93, 28, 28, 72, 56, 28, 56, 99, 72, 61, 78, 85, 71, 89, 74, 78, 61, 67, 67, 56, 60, 36, 33, 61, 33, 61, 83, 72, 80, 72, 77, 56, 61, 50, 61, 61, 53, 53, 56, 56, 50, 58, 61, 28, 50, 58, 33, 133, 122, 111, 117, 89, 56, 128, 100, 89, 72, 56, 28, 28, 78, 61, 72, 61, 72, 61, 72, 61, 72, 61, 72, 61, 56, 72, 56, 72, 56, 100, 89, 78, 61, 78, 61, 72, 56, 78, 61, 78, 61, 61, 53, 28, 133, 122, 111, 78, 61, 103, 67, 72, 61, 72, 56, 100, 89, 78, 61, 72, 56, 72, 56, 67, 56, 67, 56, 28, 28, 28, 28, 78, 61, 78, 61, 72, 39, 72, 39, 72, 61, 72, 61, 67, 56, 61, 33, 58, 52, 72, 61, 70, 84, 63, 63, 61, 50, 72, 56, 67, 56, 78, 61, 78, 61, 78, 61, 78, 61, 67, 56, 50, 83, 51, 28, 95, 95, 72, 72, 56, 61, 61, 56, 50, 65, 48, 72, 72, 67, 67, 56, 56, 28, 77, 61, 72, 39, 67, 56},
			// "Arial|100px|700|370"
			0x370: []int{54, 46, 61, 58, 33, 33, 72, 63, 75, 75, 33, 56, 56, 56, 33, 56, 75, 75, 75, 75, 33, 46, 72, 33, 85, 91, 47, 75, 82, 75, 93, 84, 28, 72, 72, 60, 72, 67, 61, 72, 78, 28, 72, 67, 83, 72, 64, 78, 72, 67, 75, 60, 61, 67, 82, 67, 81, 80, 28, 67, 61, 45, 61, 28, 58, 61, 61, 56, 61, 47, 46, 61, 54, 28, 56, 56, 61, 56, 45, 61, 77, 62, 52, 68, 45, 58, 72, 58, 75, 84, 28, 58, 61, 58, 84, 72, 61, 58, 77, 100, 77, 75, 84, 68, 78, 61, 72, 56, 61, 45, 75, 53, 80, 61, 99, 89, 70, 61, 70, 60, 67, 67, 64, 60, 73, 58, 51, 44, 68, 62, 56, 28, 78, 48, 48, 67, 61, 72, 83, 74, 62, 72, 72, 72, 67, 67, 89, 57, 71, 67, 28, 28, 56, 109, 106, 88, 61, 72, 62, 72, 72, 72, 72, 57, 71, 67, 90, 63, 72, 72, 61, 70, 83, 72, 78, 72, 67, 72, 61, 62, 85, 67, 73, 70, 100, 102, 87, 98, 72, 71, 103, 72, 56, 62, 61, 42, 63, 56, 71, 50, 61, 61, 50, 64, 74, 60, 61, 60, 61, 56, 49, 56, 88, 56, 61, 58, 83, 84, 73, 85, 61, 55, 85, 58, 56, 56, 61, 42, 55, 56, 28, 28, 28, 97, 91, 61, 50, 61, 56, 60, 128, 78, 87, 70, 98, 80, 67, 56, 93, 80, 81, 75, 108, 98, 63, 50, 81, 75, 78, 61, 81, 65, 81, 65, 112, 97, 81, 64, 128, 90, 128, 78, 72, 56, 58, 0, 0, 0, 0, 0, 0, 0, 72, 61, 72, 61, 67, 61, 49, 45, 57, 42, 70, 60, 90, 71, 63, 50, 61, 50, 61, 50, 61, 50, 76, 62, 72, 60, 88, 71, 113, 92, 72, 58, 72, 56, 61, 49, 56, 56, 56, 56, 67, 56, 85, 69, 70, 58, 70, 58, 70, 61, 86, 68, 86, 68, 28, 90, 71, 70, 60, 70, 64, 72, 60, 72, 60, 70, 58, 83, 74, 28, 72, 56, 72, 56, 100, 89, 67, 56, 73, 56, 73, 56, 90, 71, 63, 50, 63, 53, 72, 61, 72, 61, 78, 61, 78, 61, 78, 61, 71, 55, 62, 56, 62, 56, 62, 56, 70, 58, 57, 42, 98, 85, 57, 42, 67, 56, 67, 56},
		},
	},
	FontTypefaceCourier: {
		FontWeightNormal: {
			// "Courier|100px|400|0"
			0: []int{60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 50, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 0, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 0, 0, 0, 0, 0, 0, 0, 0, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 28, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60},
			// "Courier|100px|400|370"
			0x370: []int{60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 0, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60}, // "Courier|100px|400|370"
		},
		FontWeightBold: {
			// "Courier|100px|700|0"
			0: []int{60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 50, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 0, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 0, 0, 0, 0, 0, 0, 0, 0, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 33, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60},
			// "Courier|100px|700|370"
			0x370: []int{60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 0, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60, 60},
		},
	},
	FontTypefaceVerdana: {
		FontWeightNormal: {
			// "Verdana|100px|400|0"
			0: []int{100, 100, 100, 100, 100, 100, 100, 100, 100, 35, 35, 35, 35, 35, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 35, 39, 46, 82, 64, 108, 73, 27, 45, 45, 64, 82, 36, 45, 36, 45, 64, 64, 64, 64, 64, 64, 64, 64, 64, 64, 45, 45, 82, 82, 82, 55, 100, 68, 69, 70, 77, 63, 57, 78, 75, 42, 45, 69, 56, 84, 75, 79, 60, 79, 70, 68, 62, 73, 68, 99, 69, 62, 69, 45, 45, 45, 82, 64, 64, 60, 62, 52, 62, 60, 35, 62, 63, 27, 34, 59, 27, 97, 63, 61, 62, 62, 43, 52, 39, 63, 59, 82, 59, 59, 53, 63, 45, 63, 82, 50, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 35, 39, 64, 64, 64, 64, 45, 64, 64, 100, 55, 64, 82, 0, 100, 64, 54, 82, 54, 54, 64, 64, 64, 36, 64, 54, 55, 64, 100, 100, 100, 55, 68, 68, 68, 68, 68, 68, 98, 70, 63, 63, 63, 63, 42, 42, 42, 42, 78, 75, 79, 79, 79, 79, 79, 82, 79, 73, 73, 73, 73, 62, 61, 62, 60, 60, 60, 60, 60, 60, 96, 52, 60, 60, 60, 60, 27, 27, 27, 27, 61, 63, 61, 61, 61, 61, 61, 82, 61, 63, 63, 63, 63, 59, 62, 59, 68, 60, 68, 60, 68, 60, 70, 52, 70, 52, 70, 52, 70, 52, 77, 65, 78, 62, 63, 60, 63, 60, 63, 60, 63, 60, 63, 60, 78, 62, 78, 62, 78, 62, 78, 62, 75, 63, 75, 63, 42, 27, 42, 27, 42, 27, 42, 27, 42, 27, 87, 61, 45, 34, 69, 59, 59, 56, 27, 56, 27, 56, 30, 56, 46, 56, 28, 75, 63, 75, 63, 75, 63, 73, 75, 63, 79, 61, 79, 61, 79, 61, 107, 98, 70, 43, 70, 43, 70, 43, 68, 52, 68, 52, 68, 52, 68, 52, 62, 39, 62, 39, 62, 39, 73, 63, 73, 63, 73, 63, 73, 63, 73, 63, 73, 63, 99, 82, 62, 59, 62, 69, 53, 69, 53, 69, 53, 30, 50, 76, 57, 50, 57, 50, 67, 67, 44, 72, 81, 57, 50, 47, 61, 75, 50, 56, 64, 72, 72, 77, 25, 33, 72, 50, 28, 48, 82, 72, 50, 72, 81, 61, 92, 69, 65, 50, 56, 56, 39, 58, 34, 28, 61, 28, 61, 76, 66, 74, 72, 78, 50, 61, 44, 54, 54, 44, 44, 50, 50, 44, 42, 50, 20, 28, 25, 33, 133, 117, 94, 100, 89, 56, 111, 100, 78, 132, 124, 106, 91, 142, 124, 137, 127, 72, 50, 73, 63, 137, 127, 73, 63, 44, 72, 44, 72, 44, 89, 67, 72, 50, 141, 126, 133, 123, 72, 50, 72, 50, 54, 44, 98, 133, 117, 94, 78, 62, 95, 56, 75, 63, 68, 60, 98, 96, 79, 61, 72, 44, 72, 44, 61, 44, 61, 44, 33, 28, 33, 28, 72, 50, 72, 50, 67, 33, 67, 33, 72, 50, 72, 50, 68, 52, 62, 39, 56, 40, 139, 127, 65, 50, 60, 50, 61, 44, 72, 44, 61, 44, 72, 50, 72, 50, 72, 50, 72, 50, 72, 50, 28, 50, 32, 34, 77, 77, 72, 67, 50, 61, 61, 39, 44, 53, 40, 67, 72, 73, 61, 44, 39, 28, 70, 50, 67, 33, 72, 50},
			// "Verdana|100px|400|370"
			0x370: []int{55, 41, 61, 44, 33, 33, 72, 54, 100, 100, 33, 44, 44, 44, 45, 39, 100, 100, 100, 100, 64, 64, 68, 45, 75, 87, 54, 100, 88, 100, 75, 91, 27, 68, 69, 57, 70, 63, 69, 75, 79, 42, 69, 69, 84, 75, 65, 79, 75, 60, 100, 67, 62, 62, 82, 69, 87, 82, 42, 62, 62, 51, 63, 27, 63, 62, 62, 59, 61, 51, 46, 63, 62, 27, 59, 59, 64, 59, 50, 61, 64, 63, 51, 63, 50, 63, 79, 59, 82, 81, 27, 63, 61, 63, 81, 66, 51, 50, 72, 89, 72, 53, 71, 56, 72, 50, 67, 42, 56, 45, 58, 45, 73, 55, 83, 78, 62, 52, 66, 44, 54, 54, 66, 58, 70, 51, 48, 39, 56, 51, 44, 28, 72, 40, 40, 56, 50, 67, 89, 63, 50, 67, 67, 67, 63, 63, 79, 57, 70, 68, 42, 42, 45, 112, 110, 82, 69, 75, 62, 75, 68, 69, 69, 57, 75, 63, 97, 62, 75, 75, 69, 73, 84, 75, 79, 75, 60, 70, 62, 62, 82, 69, 76, 71, 103, 104, 78, 92, 68, 70, 103, 71, 60, 61, 59, 47, 62, 60, 80, 52, 64, 64, 59, 62, 70, 64, 61, 64, 62, 53, 50, 59, 84, 59, 64, 61, 88, 89, 64, 79, 57, 55, 84, 60, 60, 60, 63, 47, 55, 52, 27, 27, 34, 91, 91, 63, 59, 64, 59, 64, 117, 63, 67, 54, 97, 68, 72, 59, 103, 83, 90, 69, 121, 94, 50, 40, 74, 63, 72, 50, 81, 59, 81, 59, 119, 104, 76, 57, 98, 81, 117, 63, 67, 44, 33, 0, 0, 0, 0, 64, 0, 0, 72, 54, 57, 47, 56, 50, 57, 47, 57, 47, 63, 52, 97, 80, 50, 40, 69, 59, 69, 59, 67, 49, 79, 57, 75, 64, 85, 62, 103, 79, 79, 62, 67, 44, 62, 50, 62, 59, 62, 59, 69, 59, 91, 75, 65, 50, 71, 61, 71, 63, 88, 69, 88, 69, 33, 161, 143, 67, 52, 68, 50, 72, 54, 72, 54, 65, 50, 89, 63, 28, 132, 124, 132, 124, 89, 67, 127, 123, 75, 60, 139, 123, 161, 143, 125, 116, 50, 44, 72, 54, 139, 128, 142, 124, 79, 61, 142, 124, 134, 118, 71, 50, 125, 123, 125, 123, 135, 124, 57, 47, 156, 143, 58, 41, 72, 50, 72, 50},
		},
		FontWeightBold: {
			// "Verdana|100px|700|0"
			0: []int{100, 100, 100, 100, 100, 100, 100, 100, 100, 34, 34, 34, 34, 34, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 34, 40, 59, 87, 71, 127, 86, 33, 54, 54, 71, 87, 36, 48, 36, 69, 71, 71, 71, 71, 71, 71, 71, 71, 71, 71, 40, 40, 87, 87, 87, 62, 96, 78, 76, 72, 83, 68, 65, 81, 84, 55, 56, 77, 64, 95, 85, 85, 73, 85, 78, 71, 68, 81, 76, 113, 76, 74, 69, 54, 69, 54, 87, 71, 71, 67, 70, 59, 70, 66, 42, 70, 71, 34, 40, 67, 34, 106, 71, 69, 70, 70, 50, 59, 46, 71, 65, 98, 67, 65, 60, 71, 54, 71, 87, 50, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 34, 40, 71, 71, 71, 71, 54, 71, 71, 96, 60, 85, 87, 0, 96, 71, 59, 87, 60, 60, 71, 72, 71, 36, 71, 60, 60, 85, 118, 118, 118, 62, 78, 78, 78, 78, 78, 78, 109, 72, 68, 68, 68, 68, 55, 55, 55, 55, 83, 85, 85, 85, 85, 85, 85, 87, 85, 81, 81, 81, 81, 74, 73, 71, 67, 67, 67, 67, 67, 67, 102, 59, 66, 66, 66, 66, 34, 34, 34, 34, 68, 71, 69, 69, 69, 69, 69, 87, 69, 71, 71, 71, 71, 65, 70, 65, 78, 67, 78, 67, 78, 67, 72, 59, 72, 59, 72, 59, 72, 59, 83, 88, 83, 70, 68, 66, 68, 66, 68, 66, 68, 66, 68, 66, 81, 70, 81, 70, 81, 70, 81, 70, 84, 71, 84, 71, 55, 34, 55, 34, 55, 34, 55, 34, 55, 34, 101, 73, 56, 40, 77, 67, 67, 64, 34, 64, 34, 64, 52, 64, 56, 64, 35, 85, 71, 85, 71, 85, 71, 83, 85, 71, 85, 69, 85, 69, 85, 69, 114, 107, 78, 50, 78, 50, 78, 50, 71, 59, 71, 59, 71, 59, 71, 59, 68, 46, 68, 47, 68, 46, 81, 71, 81, 71, 81, 71, 81, 71, 81, 71, 81, 71, 113, 98, 74, 65, 74, 69, 60, 69, 60, 69, 60, 34, 56, 75, 66, 56, 66, 56, 72, 72, 44, 72, 82, 66, 56, 52, 67, 81, 53, 61, 71, 78, 72, 78, 31, 39, 78, 56, 28, 49, 94, 72, 56, 78, 91, 69, 106, 77, 69, 56, 61, 56, 39, 65, 51, 33, 67, 33, 67, 85, 74, 80, 65, 79, 50, 67, 44, 58, 58, 46, 45, 50, 50, 38, 33, 60, 22, 32, 28, 33, 139, 117, 100, 117, 100, 61, 122, 106, 89, 149, 138, 126, 105, 156, 140, 152, 142, 72, 56, 81, 71, 152, 142, 81, 71, 44, 72, 50, 72, 50, 100, 72, 78, 50, 152, 141, 148, 138, 78, 50, 78, 50, 58, 46, 111, 139, 117, 100, 81, 70, 97, 65, 85, 71, 78, 67, 109, 102, 85, 69, 72, 50, 72, 50, 67, 44, 67, 44, 39, 28, 39, 28, 78, 50, 78, 50, 72, 44, 72, 44, 72, 56, 72, 56, 71, 59, 68, 46, 55, 37, 155, 142, 73, 56, 68, 50, 67, 44, 72, 50, 67, 44, 78, 50, 78, 50, 78, 50, 78, 50, 72, 50, 28, 56, 36, 40, 82, 82, 72, 72, 50, 67, 67, 39, 44, 53, 47, 67, 72, 72, 67, 44, 50, 33, 82, 56, 72, 44, 72, 50},
			// "Verdana|100px|700|370"
			0x370: []int{61, 48, 70, 52, 33, 33, 78, 58, 100, 100, 33, 44, 44, 44, 40, 50, 100, 100, 100, 100, 71, 71, 80, 40, 85, 100, 71, 100, 97, 100, 94, 97, 34, 78, 76, 64, 81, 68, 69, 84, 85, 55, 77, 78, 95, 85, 71, 85, 84, 73, 100, 68, 68, 74, 95, 76, 98, 84, 55, 74, 70, 58, 71, 34, 71, 70, 72, 65, 69, 58, 55, 71, 70, 34, 67, 65, 72, 65, 58, 69, 72, 70, 56, 73, 54, 71, 91, 64, 94, 89, 34, 71, 69, 71, 89, 78, 53, 60, 72, 92, 72, 62, 71, 58, 78, 50, 72, 42, 61, 55, 60, 46, 75, 55, 87, 83, 73, 59, 74, 44, 63, 59, 68, 61, 72, 52, 54, 45, 58, 50, 44, 33, 78, 42, 42, 61, 56, 72, 94, 68, 54, 72, 72, 72, 68, 68, 91, 64, 74, 71, 55, 55, 56, 122, 121, 94, 77, 85, 74, 84, 78, 76, 76, 64, 84, 68, 112, 71, 85, 85, 77, 85, 95, 84, 85, 84, 73, 72, 68, 74, 95, 76, 85, 79, 116, 118, 91, 106, 76, 74, 120, 79, 67, 70, 68, 53, 69, 66, 100, 59, 72, 72, 67, 71, 83, 72, 69, 72, 70, 60, 54, 65, 97, 67, 73, 68, 100, 101, 74, 94, 65, 61, 99, 68, 66, 66, 71, 53, 61, 59, 34, 34, 40, 101, 102, 71, 67, 72, 65, 72, 125, 62, 76, 60, 101, 66, 72, 50, 110, 77, 99, 73, 136, 100, 53, 40, 78, 69, 78, 50, 82, 60, 82, 60, 135, 123, 82, 59, 105, 90, 125, 62, 72, 44, 34, 0, 0, 0, 0, 64, 0, 0, 78, 58, 66, 53, 61, 56, 64, 53, 64, 53, 72, 58, 112, 100, 53, 40, 77, 67, 77, 67, 73, 58, 83, 63, 84, 72, 92, 67, 111, 84, 87, 69, 72, 44, 68, 54, 74, 65, 74, 65, 76, 67, 99, 82, 73, 56, 79, 68, 79, 71, 101, 79, 101, 79, 39, 183, 171, 74, 59, 75, 56, 78, 58, 78, 58, 73, 56, 94, 68, 28, 149, 138, 149, 138, 100, 72, 139, 138, 81, 66, 152, 138, 183, 171, 142, 130, 53, 46, 78, 58, 156, 143, 156, 140, 85, 69, 156, 140, 145, 132, 73, 50, 145, 136, 145, 136, 150, 140, 64, 53, 177, 165, 64, 45, 72, 50, 72, 50},
		},
	},
}

const FallbackAsciiCode int = 88 // 'X'

func getCharacterWidth100(cCode int, typeface FontTypeface, weight FontWeight) float64 {
	if weightMap, ok := sizeMap[typeface]; ok {
		if rangeMap, ok := weightMap[weight]; ok {
			if cCode < len(rangeMap[0]) {
				return float64(rangeMap[0][cCode])
			} else if cCode >= 0x370 && cCode < 0x370+len(rangeMap[0x370]) {
				return float64(rangeMap[0x370][cCode])
			} else {
				return float64(rangeMap[0][FallbackAsciiCode])
			}
		}
	}

	return float64(sizeMap[FontTypefaceCourier][FontWeightNormal][0][FallbackAsciiCode])
}

/*
func getTextRunWidth(s string, typeface FontTypeface, weight FontWeight, fontSizeInPixels float64) float64 {
	w := 0.0
	for _, c := range s {
		w += float64(getCharacterWidth100(int(c), typeface, weight)*fontSizeInPixels) / 100.0
	}
	return w
}

func getTextDimensions(s string, typeface FontTypeface, weight FontWeight, fontSizeInPixels float64) (float64, float64) {
	runs := strings.Split(s, "\n")
	w := 0.0
	h := 0.0
	for _, r := range runs {
		h += float64(fontSizeInPixels)
		runWidth := getTextRunWidth(r, typeface, weight, fontSizeInPixels)
		if w < runWidth {
			w = runWidth
		}
	}
	return w, h
}
*/

func getTextRunWidth(s string, startIdx int, endIdx int, typeface FontTypeface, weight FontWeight, fontSizeInPixels float64) float64 {
	w := 0.0
	idx := startIdx
	for idx < endIdx {
		w += float64(getCharacterWidth100(int(s[idx]), typeface, weight)*fontSizeInPixels) / 100.0
		idx++
	}
	return w
}

func getTextDimensions(s string, typeface FontTypeface, weight FontWeight, fontSizeInPixels float64, fontInterval float64) (float64, float64) {
	w := 0.0
	h := 0.0
	runStartIdx := -1
	lineCount := 0
	for curIdx, c := range s {
		if c == '\n' {
			if runStartIdx > -1 && curIdx-runStartIdx > 1 {
				// Calculate run width for: from runStartIdx to curIdx-1 including
				runWidth := getTextRunWidth(s, runStartIdx, curIdx, typeface, weight, fontSizeInPixels)
				if runWidth > w {
					w = runWidth
				}
			}
			lineCount++
			h += float64(fontSizeInPixels)
			runStartIdx = -1
		} else {
			if runStartIdx == -1 {
				runStartIdx = curIdx
			}
		}
	}

	if runStartIdx > -1 && len(s)-runStartIdx >= 1 {
		runWidth := getTextRunWidth(s, runStartIdx, len(s), typeface, weight, fontSizeInPixels)
		if runWidth > w {
			w = runWidth
		}
	}
	if w > 0.0 {
		lineCount++
		h += float64(fontSizeInPixels)
	} else {
		h = 0.0
	}

	if h > 0 {
		h += float64(lineCount-1) * fontSizeInPixels * fontInterval
	}
	return w, h
}

func getLabelDimensionsFromTextDimensions(w float64, h float64, widthExtra float64, heightExtra float64) (float64, float64) {
	labelWidth := 0.0
	if w > 0.0 {
		labelWidth = w + widthExtra
	}
	labelHeight := h
	if labelHeight > 0.0 {
		labelHeight = h + heightExtra
	}
	return labelWidth, labelHeight
}
