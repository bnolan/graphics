#version 410 core

out vec4 FragColor;

in vec2 TexCoords;

uniform sampler2D hdrBuffer;

void main()
{
    const float gamma = 2.2;
    const float exposure = 0.5;
    vec3 hdrColor = texture(hdrBuffer, TexCoords).rgb;

    // Exposure tone mapping
    vec3 mapped = vec3(1.0) - exp(-hdrColor * exposure);
    // Gamma correction
    mapped = pow(mapped, vec3(1.0 / gamma));

    FragColor = vec4(mapped, 1.0);
}
