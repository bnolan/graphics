#version 410 core

out vec4 FragColor;

in vec2 TexCoords;

uniform sampler2D x_filterTexture;

float near = 2;
float far  = 15.0;

float LinearizeDepth(float depth)
{
    return (2.0 * near) / (far + near - depth * (far - near));
}

void main()
{
    float depth = texture(x_filterTexture, TexCoords).r;
    depth = (depth);

    FragColor = vec4(vec3(depth), 1.0);
}
