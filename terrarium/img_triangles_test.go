package terrarium

import "testing"

// triangles is drawn pixel by pixel in internal/pixels (tested there);
// these check the builtin: its two forms, its options and its errors.
func TestTriangles(t *testing.T) {
	square := `let m = img.new(4, 4)
let near = [[(0, 0, 0.2), (4, 0, 0.2), (4, 4, 0.2), #ff0000], [(0, 0, 0.2), (4, 4, 0.2), (0, 4, 0.2), #ff0000]]
let far = [[(0, 0, 0.6), (4, 0, 0.6), (4, 4, 0.6), #0000ff], [(0, 0, 0.6), (4, 4, 0.6), (0, 4, 0.6), #0000ff]]
`
	check(t, []testCase{
		{"one color", `let m = img.new(2, 2); img.triangles(m, [[(0, 0), (3, 0), (0, 3), #ff0000]]); img.pixels(m)`,
			"[#ff0000, #ff0000, #ff0000, #00000000]", false},
		{"corner colors", `let m = img.new(4, 4); img.triangles(m, [[(0, 0), #ff0000, (4, 0), #00ff00, (0, 4), #0000ff]]); img.get(m, 1, 1)`,
			"#406060", false},
		{"nearer wins", square + `img.triangles(m, near); img.triangles(m, far); img.get(m, 1, 1)`, "#ff0000", false},
		{"no depth test", square + `img.triangles(m, near); img.triangles(m, far, {"depth": false}); img.get(m, 1, 1)`, "#0000ff", false},
		{"clear the depth", square + `img.triangles(m, near); img.triangles(m, far, {"clear_depth": true}); img.get(m, 1, 1)`, "#0000ff", false},
		{"an empty list", `img.triangles(img.new(1, 1), [])`, "null", false},

		{"not a triangle", `img.triangles(img.new(1, 1), [[(0, 0), (1, 0), #ff0000]])`,
			"triangle 0 must be [a, b, c, color] or [a, color, b, color, c, color] with points (x, y, z), got [(0, 0), (1, 0), #ff0000]", true},
		{"a corner is not a point", `img.triangles(img.new(1, 1), [[(0, 0), 1, (0, 1), #ff0000]])`, "triangle 0 must be", true},
		{"a missing corner color", `img.triangles(img.new(1, 1), [[(0, 0), #ff0000, (1, 0), 5, (0, 1), #ff0000]])`, "triangle 0 must be", true},
		{"unknown option", `img.triangles(img.new(1, 1), [], {"fast": true})`, `unknown option "fast" (the options are clear_depth, depth)`, true},
		{"option not a boolean", `img.triangles(img.new(1, 1), [], {"depth": 1})`, "the option depth must be true or false, got INTEGER", true},
	})
}
