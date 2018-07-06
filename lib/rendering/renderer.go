package rendering

import (
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/stojg/graphics/lib/components"
	"github.com/stojg/graphics/lib/debug"
	"github.com/stojg/graphics/lib/rendering/debugger"
	"github.com/stojg/graphics/lib/rendering/framebuffer"
	"github.com/stojg/graphics/lib/rendering/postprocess"
	"github.com/stojg/graphics/lib/rendering/shader"
	"github.com/stojg/graphics/lib/rendering/shadow"
	"github.com/stojg/graphics/lib/rendering/standard"
	"github.com/stojg/graphics/lib/rendering/technique"
	"github.com/stojg/graphics/lib/rendering/terrain"
)

func NewEngine(width, height int, logger components.Logger) *Engine {

	// @todo add more output
	var nrAttributes int32
	gl.GetIntegerv(gl.MAX_VERTEX_ATTRIBS, &nrAttributes)
	logger.Printf("No vertex attributes supported: %d\n", nrAttributes)
	if glfw.ExtensionSupported("GL_EXT_texture_filter_anisotropic") {
		var t float32
		gl.GetFloatv(gl.MAX_TEXTURE_MAX_ANISOTROPY, &t)
		logger.Printf("Anisotropy supported with %0.0f levels\n", t)
	} else {
		logger.Println("Anisotropy not supported")
	}

	if glfw.ExtensionSupported("GL_KHR_debug") {
		logger.Println("GL_KHR_debug supported")
	} else {
		logger.Println("GL_KHR_debug not supported")
	}

	gl.ClearColor(0.01, 0.01, 0.01, 1)

	gl.FrontFace(gl.CCW)
	gl.CullFace(gl.BACK)
	gl.Enable(gl.CULL_FACE)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	gl.Enable(gl.TEXTURE_CUBE_MAP_SEAMLESS)
	gl.Disable(gl.FRAMEBUFFER_SRGB)

	e := &Engine{
		width:               int32(width),
		height:              int32(height),
		state:               NewRenderState(),
		nullShader:          shader.NewShader("filter_null"),
		overlayShader:       shader.NewShader("filter_overlay"),
		multiSampledTexture: framebuffer.NewMultiSampledTexture(gl.COLOR_ATTACHMENT0, width, height, gl.RGBA16F, gl.RGBA, gl.FLOAT, gl.LINEAR, false),
	}

	e.state.AddSamplerSlot("albedo")
	e.state.AddSamplerSlot("metallic")
	e.state.AddSamplerSlot("roughness")
	e.state.AddSamplerSlot("normal")

	e.standardRenderer = standard.NewRenderer(e.state)
	e.shadowMap = shadow.NewRenderer(e.state)
	e.terrainRenderer = terrain.NewRenderer(e.state)
	e.postprocess = postprocess.New(e.state)

	e.skybox = technique.NewSkyBox("res/textures/sky0016.hdr", e.state)

	debugger.New(width, height)

	e.state.SetInteger("x_enable_env_map", 1)
	e.state.SetInteger("x_enable_skybox", 1)

	debug.CheckForError("rendering.NewEngine end")

	return e
}

type Engine struct {
	width, height    int32
	state            components.RenderState
	nullShader       *shader.Shader
	overlayShader    *shader.Shader
	standardRenderer *standard.Renderer
	shadowMap        *shadow.Renderer
	terrainRenderer  *terrain.Renderer
	postprocess      *postprocess.Renderer
	skybox           *technique.SkyBox

	multiSampledTexture *framebuffer.Texture

	fullScreenTemp *framebuffer.Texture
}

func (e *Engine) State() components.RenderState {
	return e.state
}

func (e *Engine) Render(object, terrains components.Renderable) {
	if e.state.Camera() == nil {
		panic("Camera not found, the game cannot render")
	}
	debugger.Clear()

	// update all necessary UBOs etc
	e.state.Update()

	// @todo maybe only do this every other frame?
	e.shadowMap.Render(object, terrains)

	e.multiSampledTexture.BindFrameBuffer()
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
	e.shadowMap.Load()
	e.skybox.Load()
	e.terrainRenderer.Render(terrains)
	e.standardRenderer.Render(object)
	e.skybox.Render()
	e.multiSampledTexture.UnbindFrameBuffer()

	e.postprocess.Render(e.multiSampledTexture, false)

	//e.applyFilter(e.overlayShader, debugger.Texture(), nil)
	debug.CheckForError("renderer.Engine.Draw [end]")
}