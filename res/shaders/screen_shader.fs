#version 410 core

out vec4 FragColor;

in vec2 TexCoords;

uniform sampler2D screenTexture;

void main()
{
    float value = texture(screenTexture, TexCoords).r;
    FragColor = vec4(vec3(value), 1.0);
}
