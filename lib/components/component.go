package components

import (
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/stojg/graphics/lib/physics"
)

type Drawable interface {
	Draw()
}

type Shader interface {
	Bind()
	UpdateUniforms(*physics.Transform, Material, RenderingEngine)
}

type Transformable interface {
	Transform() *physics.Transform
}

type Light interface {
	Shader() Shader
	Color() mgl32.Vec3
	Position() mgl32.Vec3
}

type PointLight interface {
	Light
	Exponent() float32
	Linear() float32
	Constant() float32
}

type Spotlight interface {
	PointLight
	Direction() mgl32.Vec3
	Cutoff() float32
}

type RenderingEngine interface {
	AddLight(light Light)
	AddCamera(camera *Camera)
	GetMainCamera() *Camera
	GetActiveLight() Light
	GetSamplerSlot(string) uint32
}

type Engine interface {
	GetRenderingEngine() RenderingEngine
}

type Component interface {
	Update(time.Duration)
	Input(time.Duration)
	Render(Shader, RenderingEngine)
	AddToEngine(Engine)
	SetParent(Transformable)
}

type GameComponent struct {
	parent Transformable
}

func (m *GameComponent) SetParent(parent Transformable) {
	m.parent = parent
}

func (m *GameComponent) Parent() Transformable {
	return m.parent
}

func (m *GameComponent) Transform() *physics.Transform {
	return m.parent.Transform()
}

func (m *GameComponent) AddToEngine(engine Engine) {
}

func (m *GameComponent) Render(Shader, RenderingEngine) {}
func (m *GameComponent) Input(time.Duration)            {}
func (m *GameComponent) Update(time.Duration)           {}
