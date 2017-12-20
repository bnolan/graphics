package rendering

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/stojg/graphics/lib/components"
	"github.com/stojg/graphics/lib/debug"
	"github.com/stojg/graphics/lib/rendering/debugger"
	"github.com/stojg/graphics/lib/rendering/framebuffer"
	"github.com/stojg/graphics/lib/rendering/primitives"
	"github.com/stojg/graphics/lib/rendering/shader"
	"github.com/stojg/graphics/lib/rendering/technique"
)

func NewEngine(width, height int) *Engine {

	var nrAttributes int32
	gl.GetIntegerv(gl.MAX_VERTEX_ATTRIBS, &nrAttributes)
	fmt.Printf("maximum nr of vertex attributes supported: %d\n", nrAttributes)

	gl.ClearColor(0.01, 0.01, 0.01, 1)

	gl.FrontFace(gl.CCW)
	gl.CullFace(gl.BACK)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	gl.Enable(gl.TEXTURE_CUBE_MAP_SEAMLESS)
	gl.Disable(gl.MULTISAMPLE)
	gl.Disable(gl.FRAMEBUFFER_SRGB)

	samplerMap := make(map[string]uint32)
	samplerMap["albedo"] = 0
	samplerMap["metallic"] = 1
	samplerMap["roughness"] = 2
	samplerMap["normal"] = 3
	samplerMap["x_shadowMap"] = 9
	samplerMap["x_filterTexture"] = 10

	e := &Engine{
		width:  int32(width),
		height: int32(height),

		samplerMap: samplerMap,

		textures:      make(map[string]components.Texture),
		uniforms3f:    make(map[string]mgl32.Vec3),
		uniformsI:     make(map[string]int32),
		uniformsFloat: make(map[string]float32),

		nullShader:    shader.NewShader("filter_null"),
		overlayShader: shader.NewShader("filter_overlay"),
		fxaaShader:    shader.NewShader("filter_fxaa"),
		gaussShader:   shader.NewShader("filter_gauss"),
		ambientShader: shader.NewShader("forward_ambient"),
		shadowShader:  shader.NewShader("shadow_vsm"),
		lightShader:   shader.NewShader("pbr_light"),

		offScreenTexture: framebuffer.NewTexture(gl.COLOR_ATTACHMENT0, width, height, gl.RGBA16F, gl.RGBA, gl.FLOAT, gl.LINEAR, false),
		toneMapShader:    shader.NewShader("filter_tonemap"),

		fullScreenTemp: framebuffer.NewTexture(gl.COLOR_ATTACHMENT0, width, height, gl.RGB, gl.RGB, gl.UNSIGNED_BYTE, gl.NEAREST, false),

		capabilities: make(map[string]bool),
	}

	envMap := framebuffer.NewHDRCubeMap(1024, 1024, "res/textures/sky0016.hdr")

	e.skybox = technique.NewSkyBox(envMap)

	irradianceMap := framebuffer.NewCubeMap(32, 32, false)
	technique.Convolute(envMap, irradianceMap)
	samplerMap["x_irradianceMap"] = 11
	e.SetTexture("x_irradianceMap", irradianceMap)

	preFilterMap := framebuffer.NewCubeMap(128, 128, true)
	technique.Prefilter(envMap, preFilterMap)
	samplerMap["x_prefilterMap"] = 12
	e.SetTexture("x_prefilterMap", preFilterMap)

	brdfLutTexture := technique.BrdfLutTexture()
	samplerMap["x_brdfLUT"] = 13
	e.SetTexture("x_brdfLUT", brdfLutTexture)

	e.shadowTextures = make([]components.Texture, 12)
	e.tempShadowTextures = make([]components.Texture, 12)
	for i := uint(0); i < 12; i++ {
		size := 1 << i // power of two, 1, 2, 4, 8, 16 and so on
		e.shadowTextures[i] = framebuffer.NewTexture(gl.COLOR_ATTACHMENT0, size, size, gl.RG32F, gl.RG, gl.FLOAT, gl.LINEAR, true)
		e.tempShadowTextures[i] = framebuffer.NewTexture(gl.COLOR_ATTACHMENT0, size, size, gl.RG32F, gl.RG, gl.FLOAT, gl.LINEAR, true)
	}

	debugger.New(width, height)

	// set defaults
	e.SetFloat("x_varianceMin", 0.0)
	e.SetFloat("x_lightBleedReductionAmount", 0.0)
	e.SetTexture("x_shadowMap", e.shadowTextures[0])

	e.SetInteger("x_enable_env_map", 1)
	e.SetInteger("x_enable_skybox", 1)

	debug.CheckForError("rendering.NewEngine end")
	return e
}

type Engine struct {
	width, height int32
	mainCamera    components.Viewable
	lights        []components.Light
	activeLight   components.Light

	skybox *technique.SkyBox

	samplerMap    map[string]uint32
	textures      map[string]components.Texture
	uniforms3f    map[string]mgl32.Vec3
	uniformsI     map[string]int32
	uniformsFloat map[string]float32

	nullShader    *shader.Shader
	gaussShader   *shader.Shader
	ambientShader *shader.Shader
	toneMapShader *shader.Shader
	shadowShader  *shader.Shader
	fxaaShader    *shader.Shader
	overlayShader *shader.Shader
	lightShader   *shader.Shader

	offScreenTexture *framebuffer.Texture

	shadowTextures     []components.Texture
	tempShadowTextures []components.Texture

	fullScreenTemp *framebuffer.Texture

	capabilities map[string]bool
}

func (e *Engine) ActiveLight() components.Light {
	return e.activeLight
}

func (e *Engine) Enable(cap string) {
	e.capabilities[cap] = true
}

func (e *Engine) Disable(cap string) {
	e.capabilities[cap] = false
}

func (e *Engine) Render(object components.Renderable) {
	if e.mainCamera == nil {
		panic("mainCamera not found, the game cannot render")
	}
	gl.Enable(gl.DEPTH_TEST)

	debugger.Clear()
	//gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	//gl.Viewport(0, 0, e.width, e.height)
	for _, light := range e.lights {
		if !light.ShadowCaster() {
			continue
		}
		e.activeLight = light
		light.SetCamera(e.MainCamera().Pos(), e.mainCamera.Rot())
		idx := light.ShadowInfo().SizeAsPowerOfTwo()
		e.shadowTextures[idx].BindAsRenderTarget()
		e.shadowTextures[idx].SetViewPort()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		object.RenderAll(e.shadowShader, e)
		debugger.AddTexture(e.shadowTextures[idx], "shadow", e.applyFilter)

	}

	e.offScreenTexture.BindAsRenderTarget()
	e.offScreenTexture.SetViewPort()
	//gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	//gl.Viewport(0, 0, e.width, e.height)

	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	object.RenderAll(e.lightShader, e)
	debugger.AddTexture(e.offScreenTexture, "rgb", e.applyFilter)

	if e.Integer("x_enable_skybox") == 1 {
		e.skybox.Draw(e)
	}

	gl.Viewport(0, 0, e.width, e.height)
	gl.Disable(gl.DEPTH_TEST)
	e.applyFilter(e.toneMapShader, e.offScreenTexture, e.fullScreenTemp)
	e.applyFilter(e.fxaaShader, e.fullScreenTemp, nil)
	e.applyFilter(e.overlayShader, debugger.Texture(), nil)
	debug.CheckForError("renderer.Engine.Render [end]")
}

func (e *Engine) Lights() []components.Light {
	return e.lights
}

func (e *Engine) AddLight(l components.Light) {
	e.lights = append(e.lights, l)
}

func (e *Engine) blurShadowMap(sizeAsPowerOfTwo int, blurAmount float32) {
	var size = 2 << uint(sizeAsPowerOfTwo)
	src := e.shadowTextures[sizeAsPowerOfTwo]
	tmp := e.tempShadowTextures[sizeAsPowerOfTwo]
	gl.Disable(gl.DEPTH_TEST)
	gl.Viewport(0, 0, src.Width(), src.Height())
	e.SetVector3f("x_blurScale", mgl32.Vec3{1 / float32(size) * blurAmount, 0, 0})
	e.applyFilter(e.gaussShader, src, tmp)
	e.SetVector3f("x_blurScale", mgl32.Vec3{0, 1 / float32(size) * blurAmount, 0})
	e.applyFilter(e.gaussShader, tmp, src)
	gl.GenerateMipmap(gl.TEXTURE_2D)
	gl.Enable(gl.DEPTH_TEST)
}

func (e *Engine) applyFilter(filter components.Shader, in, out components.Texture) {
	if in == out {
		panic("Argh, can't apply filter where source and destination is the same")
	}

	if out == nil {
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	} else {
		out.BindAsRenderTarget()
	}
	e.SetTexture("x_filterTexture", in)
	e.SetInteger("x_w", in.Width())
	e.SetInteger("x_h", in.Height())
	filter.Bind()
	filter.UpdateUniforms(nil, nil, e)
	primitives.DrawQuad()
}

func (e *Engine) SetTexture(name string, texture components.Texture) {
	e.textures[name] = texture
}

func (e *Engine) Texture(name string) components.Texture {
	v, ok := e.textures[name]
	if !ok {
		panic(fmt.Sprintf("Texture, Could not find texture '%s'\n", name))
	}
	return v
}

func (e *Engine) SetInteger(name string, v int32) {
	e.uniformsI[name] = v
}

func (e *Engine) Integer(name string) int32 {
	v, ok := e.uniformsI[name]
	if !ok {
		panic(fmt.Sprintf("Integer, no value found for uniform '%s'", name))
	}
	return v
}

func (e *Engine) SetFloat(name string, v float32) {
	e.uniformsFloat[name] = v
}

func (e *Engine) Float(name string) float32 {
	v, ok := e.uniformsFloat[name]
	if !ok {
		panic(fmt.Sprintf("Float, no value found for uniform '%s'", name))
	}
	return v
}

func (e *Engine) SetVector3f(name string, v mgl32.Vec3) {
	e.uniforms3f[name] = v
}

func (e *Engine) Vector3f(name string) mgl32.Vec3 {
	// @todo set value, regardless, this might be an array that isn't used
	v := e.uniforms3f[name]
	//if !ok {
	//	fmt.Printf("Vector3f, no value found for uniform '%s'\n", name)
	//}
	return v
}

func (e *Engine) AddCamera(c components.Viewable) {
	e.mainCamera = c
}

func (e *Engine) MainCamera() components.Viewable {
	return e.mainCamera
}

func (e *Engine) SamplerSlot(samplerName string) uint32 {
	slot, exists := e.samplerMap[samplerName]
	if !exists {
		fmt.Printf("rendering.Engine tried finding texture slot for %s, failed\n", samplerName)
	}
	return slot
}

func (e *Engine) SetSamplerSlot(samplerName string, slot uint32) {
	e.samplerMap[samplerName] = slot
}
