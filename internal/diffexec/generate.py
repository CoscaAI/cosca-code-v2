#!/usr/bin/env python3
"""Gerador de imagem por diffusion (SD 1.5 + ROCm).
Recebe prompt + args via argv e salva a imagem em out_path.
Chamado pelo diffexec (Go) como subprocesso."""
import sys
import time
import torch
from diffusers import StableDiffusionPipeline

def main():
    args = sys.argv[1:]
    out_path = args[0]
    prompt = args[1]
    steps = int(args[2]) if len(args) > 2 else 25
    cfg = float(args[3]) if len(args) > 3 else 7.5

    pipe = StableDiffusionPipeline.from_pretrained(
        '/home/cosca/.cosca/models/sd15',
        torch_dtype=torch.float16,
        safety_checker=None,
    )
    pipe = pipe.to('cuda')
    pipe.enable_attention_slicing()

    t0 = time.time()
    img = pipe(prompt, num_inference_steps=steps, guidance_scale=cfg).images[0]
    img.save(out_path)
    print(json_result({'ok': True, 'path': out_path, 'seconds': round(time.time()-t0, 1), 'size': img.size}))

def json_result(d):
    import json
    return json.dumps(d)

if __name__ == '__main__':
    main()
