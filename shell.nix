{ pkgs ? import <nixpkgs> {} }:
let
    xorgLibs = with pkgs.xorg;[
        libXi
        libXxf86vm
        libX11.dev
        libXft
        libXcursor
        libXinerama
        libXrandr
        libXrender
        xorgproto
    ];
in
pkgs.mkShell{
    buildInputs = with pkgs; [
        glfw

        pkg-config
        libcap go gcc
        libGL
        libGLU 
        xorgLibs
    ];
}
