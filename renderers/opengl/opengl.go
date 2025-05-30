//go:build cgo

package opengl

import (
	"fmt"
	"image"
	"strings"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

// OpenGL is an open graphics library renderer.
type OpenGL struct {
	*rasterizer.Rasterizer

	img     *image.RGBA
	program uint32
	vao     uint32
	texture uint32
}

// New returns an open graphics library (OpenGL) renderer.
func New(width, height float64, resolution canvas.Resolution) *OpenGL {
	img := image.NewRGBA(image.Rect(0, 0, int(width*resolution.DPMM()+0.5), int(height*resolution.DPMM()+0.5)))
	return &OpenGL{
		Rasterizer: rasterizer.FromImage(img, resolution, nil),
		img:        img,
	}
}

func (r *OpenGL) Compile() {
	points := []float32{
		-1.0, -1.0, 0.0, 1.0,
		1.0, -1.0, 1.0, 1.0,
		-1.0, 1.0, 0.0, 0.0,

		1.0, -1.0, 1.0, 1.0,
		1.0, 1.0, 1.0, 0.0,
		-1.0, 1.0, 0.0, 0.0,
	}

	// compile shaders
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		panic(err)
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		panic(err)
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	// generate texture
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_R, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)

	width := int32(r.img.Rect.Size().X)
	height := int32(r.img.Rect.Size().Y)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, width, height, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(r.img.Pix))
	gl.GenerateMipmap(texture)

	// create data points
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	// attach attributes
	vertexAttrib := uint32(gl.GetAttribLocation(program, gl.Str("position\x00")))
	texcoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vertTexcoord\x00")))
	texUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.EnableVertexAttribArray(vertexAttrib)
	gl.EnableVertexAttribArray(texcoordAttrib)
	gl.VertexAttribPointer(vertexAttrib, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(0))
	gl.VertexAttribPointer(texcoordAttrib, 2, gl.FLOAT, false, 4*4, gl.PtrOffset(2*4))
	gl.Uniform1i(texUniform, int32(0))

	// unbind
	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
	gl.UseProgram(0)

	r.program = program
	r.vao = vao
	r.texture = texture
}

func (r *OpenGL) Draw() {
	gl.UseProgram(r.program)
	gl.BindVertexArray(r.vao)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, r.texture)

	gl.DrawArrays(gl.TRIANGLES, 0, 2*3)

	gl.BindTexture(gl.TEXTURE_2D, 0)
	gl.BindVertexArray(0)
	gl.UseProgram(0)
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}
	return shader, nil
}

var vertexShaderSource = `
	#version 410
	in vec2 position;
	in vec2 vertTexcoord;

	out vec2 fragTexcoord;

	void main() {
		gl_Position = vec4(position, 0.0, 1.0);
		fragTexcoord = vertTexcoord;
	}
` + "\x00"

var fragmentShaderSource = `
	#version 410
	in vec2 fragTexcoord;

	out vec4 color;

	uniform sampler2D tex;

	void main() {
		color = texture(tex, fragTexcoord);
	}
` + "\x00"

//import (
//	"fmt"
//	"image/color"
//	"strings"
//
//	"github.com/go-gl/gl/v3.3-core/gl"
//)
//
//var vertexShaderSource = `
//	#version 410
//	in vec2 position;
//	in vec4 vertTexcoord;
//	in vec4 vertColor;
//
//	out vec4 fragTexcoord;
//	out vec4 fragColor;
//
//	void main() {
//		gl_Position = vec4(position, 0.0, 1.0);
//		fragTexcoord = vertTexcoord;
//		fragColor = vertColor;
//	}
//` + "\x00"
//
//var fragmentShaderSource = `
//	#version 410
//	in vec4 fragTexcoord;
//	in vec4 fragColor;
//
//	out vec4 color;
//
//	void main() {
//		float u = fragTexcoord.s;
//		float v = fragTexcoord.t;
//		float w1 = fragTexcoord.p;
//		float w2 = fragTexcoord.q;
//
//		float denom = ((1-u)*(1-u)*(1-u) + w1*(1-u)*(1-u)*u + w2*(1-u)*u*u + u*u*u);
//		float f = v - (w1*(1-u)*(1-u)*u + w2*(1-u)*u*u) / denom;
//		float gx = dFdx(fragTexcoord.st)
//		float gy = dFdy(fragTexcoord.st)
//		float g =
//		float e = 0.5 - f / sqrt(g.x*g.x+g.y*g.y);
//
//		vec2 p = fragTexcoord.st;
//		vec2 px = dFdx(p);
//		vec2 py = dFdy(p);
//		float fx = (2*p.x)*px.x - px.y;
//		float fy = (2*p.x)*py.x - py.y;
//		float sd = (p.x*p.x - p.y)/sqrt(fx*fx + fy*fy);
//
//		float alpha = 0.5 - sd;
//		if (e >= 1)
//			color = fragColor;
//		else if (e <= 0)
//			discard;
//		else
//			color = vec4(fragColor.rgb, fragColor.a*e);
//	}
//` + "\x00"
//
//func compileShader(source string, shaderType uint32) (uint32, error) {
//	shader := gl.CreateShader(shaderType)
//
//	csources, free := gl.Strs(source)
//	gl.ShaderSource(shader, 1, csources, nil)
//	free()
//	gl.CompileShader(shader)
//
//	var status int32
//	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
//	if status == gl.FALSE {
//		var logLength int32
//		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
//
//		log := strings.Repeat("\x00", int(logLength+1))
//		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
//
//		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
//	}
//
//	return shader, nil
//}
//
//type OpenGL struct {
//	points       []float32
//	program, vao uint32
//	n            int32
//}
//
//func newOpenGL() *OpenGL {
//	return &OpenGL{}
//}
//
//func (ogl *OpenGL) AddPath(p *Path, color color.RGBA) {
//	a := float32(color.A) / 255.0
//	r := float32(color.R) / 255.0 / a
//	g := float32(color.G) / 255.0 / a
//	b := float32(color.B) / 255.0 / a
//
//	triangles, beziers := p.Tessellate()
//	for _, tr := range triangles {
//		ogl.points = append(ogl.points, float32(tr[0].X), float32(tr[0].Y), 0.5, 0.0, 0.0, 0.0, r, g, b, a)
//		ogl.points = append(ogl.points, float32(tr[1].X), float32(tr[1].Y), 0.5, 0.0, 0.0, 0.0, r, g, b, a)
//		ogl.points = append(ogl.points, float32(tr[2].X), float32(tr[2].Y), 0.5, 0.0, 0.0, 0.0, r, g, b, a)
//	}
//	for _, bz := range beziers {
//		w1 := float32(bz[4].X)
//		w2 := float32(bz[4].Y)
//		ogl.points = append(ogl.points, float32(bz[0].X), float32(bz[0].Y), 0.0, 0.0, w1, w2, r, g, b, a)
//		ogl.points = append(ogl.points, float32(bz[2].X), float32(bz[2].Y), 1.0, 1.0, w1, w2, r, g, b, a)
//		ogl.points = append(ogl.points, float32(bz[1].X), float32(bz[1].Y), 0.0, 1.0, w1, w2, r, g, b, a)
//
//		ogl.points = append(ogl.points, float32(bz[3].X), float32(bz[3].Y), 1.0, 0.0, w1, w2, r, g, b, a)
//		ogl.points = append(ogl.points, float32(bz[2].X), float32(bz[2].Y), 1.0, 1.0, w1, w2, r, g, b, a)
//		ogl.points = append(ogl.points, float32(bz[0].X), float32(bz[0].Y), 0.0, 0.0, w1, w2, r, g, b, a)
//	}
//}
//
//func (ogl *OpenGL) Compile() {
//	const N = 10
//
//	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
//	if err != nil {
//		panic(err)
//	}
//	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
//	if err != nil {
//		panic(err)
//	}
//
//	prog := gl.CreateProgram()
//	gl.AttachShader(prog, vertexShader)
//	gl.AttachShader(prog, fragmentShader)
//	gl.LinkProgram(prog)
//
//	var vbo uint32
//	gl.GenBuffers(1, &vbo)
//	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
//	gl.BufferData(gl.ARRAY_BUFFER, 4*len(ogl.points), gl.Ptr(ogl.points), gl.STATIC_DRAW)
//
//	var vao uint32
//	gl.GenVertexArrays(1, &vao)
//	gl.BindVertexArray(vao)
//
//	vertexAttrib := uint32(gl.GetAttribLocation(prog, gl.Str("position\x00")))
//	texcoordAttrib := uint32(gl.GetAttribLocation(prog, gl.Str("vertTexcoord\x00")))
//	colorAttrib := uint32(gl.GetAttribLocation(prog, gl.Str("vertColor\x00")))
//	gl.EnableVertexAttribArray(vertexAttrib)
//	gl.EnableVertexAttribArray(texcoordAttrib)
//	gl.EnableVertexAttribArray(colorAttrib)
//	gl.VertexAttribPointer(vertexAttrib, 2, gl.FLOAT, false, N*4, gl.PtrOffset(0))
//	gl.VertexAttribPointer(texcoordAttrib, 4, gl.FLOAT, false, N*4, gl.PtrOffset(2*4))
//	gl.VertexAttribPointer(colorAttrib, 4, gl.FLOAT, false, N*4, gl.PtrOffset(6*4))
//
//	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
//	gl.BindVertexArray(0)
//	gl.UseProgram(0)
//
//	ogl.program = prog
//	ogl.vao = vao
//	ogl.n = int32(len(ogl.points) / N)
//}
//
//func (ogl *OpenGL) Draw() {
//	gl.UseProgram(ogl.program)
//	gl.BindVertexArray(ogl.vao)
//
//	gl.DrawArrays(gl.TRIANGLES, 0, ogl.n)
//
//	gl.BindVertexArray(0)
//	gl.UseProgram(0)
//}
//
//func (l pathLayer) ToOpenGL(ogl *OpenGL) {
//	// TODO: use fill rule (EvenOdd) for OpenGL
//	if l.fillColor.A != 0 {
//		ogl.AddPath(l.path, l.fillColor)
//	}
//	if l.strokeColor.A != 0 && 0.0 < l.strokeWidth {
//		stroke := l.path
//		if 0 < len(l.dashes) {
//			dashOffset, dashes := canvas.ScaleDash(style.StrokeWidth, l.dashOffset, l.dashes)
//			stroke = stroke.Dash(dashOffset, dashes...)
//		}
//		stroke= stroke.Stroke(l.strokeWidth, l.strokeCapper, l.strokeJoiner)
//		ogl.AddPath(stroke, l.strokeColor)
//	}
//}
//
//func (l textLayer) ToOpenGL(ogl *OpenGL) {
//	paths, colors := l.text.ToPaths()
//	for i, path := range paths {
//		state := defaultDrawState
//		state.fillColor = colors[i]
//		pathLayer{path.Transform(l.m), state, false}.ToOpenGL(ogl)
//	}
//}
//
//func (l imageLayer) ToOpenGL(ogl *OpenGL) {
//	panic("images not supported in OpenGL")
//}
