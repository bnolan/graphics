#version 410 core

layout (location = 0) in vec3 position;
layout (location = 2) in vec2 aTexCoord;

uniform mat4 projection;
uniform mat4 view;
uniform mat4 model;

out vec2 TexCoord;

void main() {
    gl_Position = projection * view * model * vec4(position, 1.0);
    TexCoord = aTexCoord;
}
