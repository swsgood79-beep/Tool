# main.(*Host).approveCombined @ 0x140793720
   140793720:	4c 8d a4 24 28 fc ff 	lea    -0x3d8(%rsp),%r12
   140793727:	ff 
   140793728:	4d 3b 66 10          	cmp    0x10(%r14),%r12
   14079372c:	0f 86 49 09 00 00    	jbe    0x14079407b
   140793732:	55                   	push   %rbp
   140793733:	48 89 e5             	mov    %rsp,%rbp
   140793736:	48 81 ec 50 04 00 00 	sub    $0x450,%rsp
   14079373d:	48 89 84 24 60 04 00 	mov    %rax,0x460(%rsp)
   140793744:	00 
   140793745:	48 89 9c 24 68 04 00 	mov    %rbx,0x468(%rsp)
   14079374c:	00 
   14079374d:	48 89 8c 24 70 04 00 	mov    %rcx,0x470(%rsp)
   140793754:	00 
   140793755:	48 8d 90 d0 00 00 00 	lea    0xd0(%rax),%rdx
   14079375c:	48 89 94 24 38 04 00 	mov    %rdx,0x438(%rsp)
   140793763:	00 
   140793764:	48 89 c6             	mov    %rax,%rsi
   140793767:	31 c0                	xor    %eax,%eax
   140793769:	41 b8 01 00 00 00    	mov    $0x1,%r8d
   14079376f:	f0 44 0f b1 86 d0 00 	lock cmpxchg %r8d,0xd0(%rsi)
   140793776:	00 00 
   140793778:	41 0f 94 c0          	sete   %r8b
   14079377c:	0f 1f 40 00          	nopl   0x0(%rax)
   140793780:	45 84 c0             	test   %r8b,%r8b
   140793783:	75 28                	jne    0x1407937ad
   140793785:	48 89 d0             	mov    %rdx,%rax
   140793788:	e8 f3 4f 90 ff       	call   0x140098780 # internal/sync.(*Mutex).lockSlow
   14079378d:	48 8b 8c 24 70 04 00 	mov    0x470(%rsp),%rcx
   140793794:	00 
   140793795:	48 8b 94 24 38 04 00 	mov    0x438(%rsp),%rdx
   14079379c:	00 
   14079379d:	48 8b 9c 24 68 04 00 	mov    0x468(%rsp),%rbx
   1407937a4:	00 
   1407937a5:	48 8b b4 24 60 04 00 	mov    0x460(%rsp),%rsi
   1407937ac:	00 
   1407937ad:	4c 8d 86 d8 00 00 00 	lea    0xd8(%rsi),%r8
   1407937b4:	48 8b be e8 01 00 00 	mov    0x1e8(%rsi),%rdi
   1407937bb:	4c 8b 8e e0 01 00 00 	mov    0x1e0(%rsi),%r9
   1407937c2:	4c 8d 94 24 68 01 00 	lea    0x168(%rsp),%r10
   1407937c9:	00 
   1407937ca:	41 bb 04 00 00 00    	mov    $0x4,%r11d
   1407937d0:	45 0f 10 30          	movups (%r8),%xmm14
   1407937d4:	45 0f 11 32          	movups %xmm14,(%r10)
   1407937d8:	45 0f 10 70 10       	movups 0x10(%r8),%xmm14
   1407937dd:	45 0f 11 72 10       	movups %xmm14,0x10(%r10)
   1407937e2:	45 0f 10 70 20       	movups 0x20(%r8),%xmm14
   1407937e7:	45 0f 11 72 20       	movups %xmm14,0x20(%r10)
   1407937ec:	45 0f 10 70 30       	movups 0x30(%r8),%xmm14
   1407937f1:	45 0f 11 72 30       	movups %xmm14,0x30(%r10)
   1407937f6:	49 83 c0 40          	add    $0x40,%r8
   1407937fa:	49 83 c2 40          	add    $0x40,%r10
   1407937fe:	66 90                	xchg   %ax,%ax
   140793800:	41 ff cb             	dec    %r11d
   140793803:	75 cb                	jne    0x1407937d0
   140793805:	45 0f 10 30          	movups (%r8),%xmm14
   140793809:	45 0f 11 32          	movups %xmm14,(%r10)
   14079380d:	45 0f 10 70 10       	movups 0x10(%r8),%xmm14
   140793812:	45 0f 11 72 10       	movups %xmm14,0x10(%r10)
   140793817:	45 0f 10 70 20       	movups 0x20(%r8),%xmm14
   14079381c:	45 0f 11 72 20       	movups %xmm14,0x20(%r10)
   140793821:	45 0f 10 70 28       	movups 0x28(%r8),%xmm14
   140793826:	45 0f 11 72 28       	movups %xmm14,0x28(%r10)
   14079382b:	48 85 ff             	test   %rdi,%rdi
   14079382e:	75 0a                	jne    0x14079383a
   140793830:	31 c0                	xor    %eax,%eax
   140793832:	45 31 c0             	xor    %r8d,%r8d
   140793835:	45 31 d2             	xor    %r10d,%r10d
   140793838:	eb 3c                	jmp    0x140793876
   14079383a:	48 89 bc 24 48 01 00 	mov    %rdi,0x148(%rsp)
   140793841:	00 
   140793842:	4c 89 8c 24 48 04 00 	mov    %r9,0x448(%rsp)
   140793849:	00 
   14079384a:	31 c0                	xor    %eax,%eax
   14079384c:	48 89 fb             	mov    %rdi,%rbx
   14079384f:	31 c9                	xor    %ecx,%ecx
   140793851:	48 8d 35 d8 1e c7 00 	lea    0xc71ed8(%rip),%rsi        # 0x141405730
   140793858:	e8 a3 54 8f ff       	call   0x140088d00 # runtime.growslice
   14079385d:	48 8b bc 24 48 01 00 	mov    0x148(%rsp),%rdi
   140793864:	00 
   140793865:	4c 8b 8c 24 48 04 00 	mov    0x448(%rsp),%r9
   14079386c:	00 
   14079386d:	49 89 c0             	mov    %rax,%r8
   140793870:	49 89 ca             	mov    %rcx,%r10
   140793873:	48 89 d8             	mov    %rbx,%rax
   140793876:	4c 89 94 24 48 01 00 	mov    %r10,0x148(%rsp)
   14079387d:	00 
   14079387e:	48 89 84 24 40 01 00 	mov    %rax,0x140(%rsp)
   140793885:	00 
   140793886:	4c 89 84 24 48 04 00 	mov    %r8,0x448(%rsp)
   14079388d:	00 
   14079388e:	4c 89 c3             	mov    %r8,%rbx
   140793891:	48 89 c1             	mov    %rax,%rcx
   140793894:	48 89 fe             	mov    %rdi,%rsi
   140793897:	48 8d 05 92 1e c7 00 	lea    0xc71e92(%rip),%rax        # 0x141405730
   14079389e:	4c 89 cf             	mov    %r9,%rdi
   1407938a1:	e8 3a 1d 8f ff       	call   0x1400855e0 # runtime.typedslicecopy
   1407938a6:	48 8b 94 24 48 04 00 	mov    0x448(%rsp),%rdx
   1407938ad:	00 
   1407938ae:	48 89 94 24 70 02 00 	mov    %rdx,0x270(%rsp)
   1407938b5:	00 
   1407938b6:	48 8b 94 24 40 01 00 	mov    0x140(%rsp),%rdx
   1407938bd:	00 
   1407938be:	48 89 94 24 78 02 00 	mov    %rdx,0x278(%rsp)
   1407938c5:	00 
   1407938c6:	48 8b 94 24 48 01 00 	mov    0x148(%rsp),%rdx
   1407938cd:	00 
   1407938ce:	48 89 94 24 80 02 00 	mov    %rdx,0x280(%rsp)
   1407938d5:	00 
   1407938d6:	48 8b bc 24 90 02 00 	mov    0x290(%rsp),%rdi
   1407938dd:	00 
   1407938de:	48 8b 94 24 88 02 00 	mov    0x288(%rsp),%rdx
   1407938e5:	00 
   1407938e6:	48 85 ff             	test   %rdi,%rdi
   1407938e9:	75 08                	jne    0x1407938f3
   1407938eb:	31 db                	xor    %ebx,%ebx
   1407938ed:	31 c0                	xor    %eax,%eax
   1407938ef:	31 c9                	xor    %ecx,%ecx
   1407938f1:	eb 33                	jmp    0x140793926
   1407938f3:	48 89 bc 24 48 01 00 	mov    %rdi,0x148(%rsp)
   1407938fa:	00 
   1407938fb:	48 89 94 24 40 04 00 	mov    %rdx,0x440(%rsp)
   140793902:	00 
   140793903:	31 c0                	xor    %eax,%eax
   140793905:	48 89 fb             	mov    %rdi,%rbx
   140793908:	31 c9                	xor    %ecx,%ecx
   14079390a:	48 8d 35 47 6a c2 00 	lea    0xc26a47(%rip),%rsi        # 0x1413ba358
   140793911:	e8 ea 53 8f ff       	call   0x140088d00 # runtime.growslice
   140793916:	48 8b 94 24 40 04 00 	mov    0x440(%rsp),%rdx
   14079391d:	00 
   14079391e:	48 8b bc 24 48 01 00 	mov    0x148(%rsp),%rdi
   140793925:	00 
   140793926:	48 89 8c 24 48 01 00 	mov    %rcx,0x148(%rsp)
   14079392d:	00 
   14079392e:	48 89 9c 24 40 01 00 	mov    %rbx,0x140(%rsp)
   140793935:	00 
   140793936:	48 89 84 24 40 04 00 	mov    %rax,0x440(%rsp)
   14079393d:	00 
   14079393e:	48 89 d9             	mov    %rbx,%rcx
   140793941:	48 89 fe             	mov    %rdi,%rsi
   140793944:	48 89 c3             	mov    %rax,%rbx
   140793947:	48 89 d7             	mov    %rdx,%rdi
   14079394a:	48 8d 05 07 6a c2 00 	lea    0xc26a07(%rip),%rax        # 0x1413ba358
   140793951:	e8 8a 1c 8f ff       	call   0x1400855e0 # runtime.typedslicecopy
   140793956:	48 8b 94 24 40 04 00 	mov    0x440(%rsp),%rdx
   14079395d:	00 
   14079395e:	48 89 94 24 88 02 00 	mov    %rdx,0x288(%rsp)
   140793965:	00 
   140793966:	48 8b 94 24 40 01 00 	mov    0x140(%rsp),%rdx
   14079396d:	00 
   14079396e:	48 89 94 24 90 02 00 	mov    %rdx,0x290(%rsp)
   140793975:	00 
   140793976:	48 8b 94 24 48 01 00 	mov    0x148(%rsp),%rdx
   14079397d:	00 
   14079397e:	48 89 94 24 98 02 00 	mov    %rdx,0x298(%rsp)
   140793985:	00 
   140793986:	48 8b 94 24 70 02 00 	mov    0x270(%rsp),%rdx
   14079398d:	00 
   14079398e:	4c 8b 84 24 78 02 00 	mov    0x278(%rsp),%r8
   140793995:	00 
   140793996:	4c 8d 8c 24 00 03 00 	lea    0x300(%rsp),%r9
   14079399d:	00 
   14079399e:	4c 8d 94 24 68 01 00 	lea    0x168(%rsp),%r10
   1407939a5:	00 
   1407939a6:	41 bb 04 00 00 00    	mov    $0x4,%r11d
   1407939ac:	45 0f 10 32          	movups (%r10),%xmm14
   1407939b0:	45 0f 11 31          	movups %xmm14,(%r9)
   1407939b4:	45 0f 10 72 10       	movups 0x10(%r10),%xmm14
   1407939b9:	45 0f 11 71 10       	movups %xmm14,0x10(%r9)
   1407939be:	45 0f 10 72 20       	movups 0x20(%r10),%xmm14
   1407939c3:	45 0f 11 71 20       	movups %xmm14,0x20(%r9)
   1407939c8:	45 0f 10 72 30       	movups 0x30(%r10),%xmm14
   1407939cd:	45 0f 11 71 30       	movups %xmm14,0x30(%r9)
   1407939d2:	49 83 c2 40          	add    $0x40,%r10
   1407939d6:	49 83 c1 40          	add    $0x40,%r9
   1407939da:	41 ff cb             	dec    %r11d
   1407939dd:	75 cd                	jne    0x1407939ac
   1407939df:	45 0f 10 32          	movups (%r10),%xmm14
   1407939e3:	45 0f 11 31          	movups %xmm14,(%r9)
   1407939e7:	45 0f 10 72 10       	movups 0x10(%r10),%xmm14
   1407939ec:	45 0f 11 71 10       	movups %xmm14,0x10(%r9)
   1407939f1:	45 0f 10 72 20       	movups 0x20(%r10),%xmm14
   1407939f6:	45 0f 11 71 20       	movups %xmm14,0x20(%r9)
   1407939fb:	45 0f 10 72 28       	movups 0x28(%r10),%xmm14
   140793a00:	45 0f 11 71 28       	movups %xmm14,0x28(%r9)
   140793a05:	45 31 c9             	xor    %r9d,%r9d
   140793a08:	4c 8b 94 24 68 04 00 	mov    0x468(%rsp),%r10
   140793a0f:	00 
   140793a10:	eb 0e                	jmp    0x140793a20
   140793a12:	48 81 c2 98 01 00 00 	add    $0x198,%rdx
   140793a19:	49 ff c1             	inc    %r9
   140793a1c:	0f 1f 40 00          	nopl   0x0(%rax)
   140793a20:	4d 39 c1             	cmp    %r8,%r9
   140793a23:	7d 0b                	jge    0x140793a30
   140793a25:	4c 39 92 b8 00 00 00 	cmp    %r10,0xb8(%rdx)
   140793a2c:	75 e4                	jne    0x140793a12
   140793a2e:	eb 10                	jmp    0x140793a40
   140793a30:	49 c7 c1 ff ff ff ff 	mov    $0xffffffffffffffff,%r9
   140793a37:	66 0f 1f 84 00 00 00 	nopw   0x0(%rax,%rax,1)
   140793a3e:	00 00 
   140793a40:	4d 85 c9             	test   %r9,%r9
   140793a43:	0f 8c cc 02 00 00    	jl     0x140793d15
   140793a49:	48 8b 94 24 10 04 00 	mov    0x410(%rsp),%rdx
   140793a50:	00 
   140793a51:	49 39 d1             	cmp    %rdx,%r9
   140793a54:	0f 83 1b 06 00 00    	jae    0x140794075
   140793a5a:	48 8b 94 24 08 04 00 	mov    0x408(%rsp),%rdx
   140793a61:	00 
   140793a62:	49 69 f1 98 01 00 00 	imul   $0x198,%r9,%rsi
   140793a69:	48 8d 3c 32          	lea    (%rdx,%rsi,1),%rdi
   140793a6d:	4c 8b 84 24 70 04 00 	mov    0x470(%rsp),%r8
   140793a74:	00 
   140793a75:	4d 85 c0             	test   %r8,%r8
   140793a78:	7e 72                	jle    0x140793aec
   140793a7a:	4c 8b 8c 32 48 01 00 	mov    0x148(%rdx,%rsi,1),%r9
   140793a81:	00 
   140793a82:	4d 39 c1             	cmp    %r8,%r9
   140793a85:	75 65                	jne    0x140793aec
   140793a87:	4c 8b 8c 32 08 01 00 	mov    0x108(%rdx,%rsi,1),%r9
   140793a8e:	00 
   140793a8f:	4d 39 c1             	cmp    %r8,%r9
   140793a92:	75 58                	jne    0x140793aec
   140793a94:	4c 8b 8c 32 78 01 00 	mov    0x178(%rdx,%rsi,1),%r9
   140793a9b:	00 
   140793a9c:	0f 1f 40 00          	nopl   0x0(%rax)
   140793aa0:	4d 85 c9             	test   %r9,%r9
   140793aa3:	74 47                	je     0x140793aec
   140793aa5:	4c 8b 8c 32 38 01 00 	mov    0x138(%rdx,%rsi,1),%r9
   140793aac:	00 
   140793aad:	4d 85 c9             	test   %r9,%r9
   140793ab0:	74 3a                	je     0x140793aec
   140793ab2:	90                   	nop
   140793ab3:	90                   	nop
   140793ab4:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   140793ab9:	48 8b 94 24 60 04 00 	mov    0x460(%rsp),%rdx
   140793ac0:	00 
   140793ac1:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   140793ac8:	00 
   140793ac9:	ff cb                	dec    %ebx
   140793acb:	74 0d                	je     0x140793ada
   140793acd:	48 8b 84 24 38 04 00 	mov    0x438(%rsp),%rax
   140793ad4:	00 
   140793ad5:	e8 86 4f 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   140793ada:	b8 c8 00 00 00       	mov    $0xc8,%eax
   140793adf:	31 db                	xor    %ebx,%ebx
   140793ae1:	31 c9                	xor    %ecx,%ecx
   140793ae3:	48 81 c4 50 04 00 00 	add    $0x450,%rsp
   140793aea:	5d                   	pop    %rbp
   140793aeb:	c3                   	ret
   140793aec:	44 0f b6 8c 32 e0 00 	movzbl 0xe0(%rdx,%rsi,1),%r9d
   140793af3:	00 00 
   140793af5:	45 84 c9             	test   %r9b,%r9b
   140793af8:	0f 84 c8 00 00 00    	je     0x140793bc6
   140793afe:	4c 8d 8c 24 68 01 00 	lea    0x168(%rsp),%r9
   140793b05:	00 
   140793b06:	41 bb 06 00 00 00    	mov    $0x6,%r11d
   140793b0c:	44 0f 10 37          	movups (%rdi),%xmm14
   140793b10:	45 0f 11 31          	movups %xmm14,(%r9)
   140793b14:	44 0f 10 77 10       	movups 0x10(%rdi),%xmm14
   140793b19:	45 0f 11 71 10       	movups %xmm14,0x10(%r9)
   140793b1e:	44 0f 10 77 20       	movups 0x20(%rdi),%xmm14
   140793b23:	45 0f 11 71 20       	movups %xmm14,0x20(%r9)
   140793b28:	44 0f 10 77 30       	movups 0x30(%rdi),%xmm14
   140793b2d:	45 0f 11 71 30       	movups %xmm14,0x30(%r9)
   140793b32:	48 83 c7 40          	add    $0x40,%rdi
   140793b36:	49 83 c1 40          	add    $0x40,%r9
   140793b3a:	41 ff cb             	dec    %r11d
   140793b3d:	75 cd                	jne    0x140793b0c
   140793b3f:	44 0f 10 37          	movups (%rdi),%xmm14
   140793b43:	45 0f 11 31          	movups %xmm14,(%r9)
   140793b47:	44 0f 10 77 08       	movups 0x8(%rdi),%xmm14
   140793b4c:	45 0f 11 71 08       	movups %xmm14,0x8(%r9)
   140793b51:	48 83 bc 24 c0 02 00 	cmpq   $0x0,0x2c0(%rsp)
   140793b58:	00 00 
   140793b5a:	74 2a                	je     0x140793b86
   140793b5c:	48 83 bc 24 e0 02 00 	cmpq   $0x0,0x2e0(%rsp)
   140793b63:	00 00 
   140793b65:	75 1f                	jne    0x140793b86
   140793b67:	48 83 bc 24 80 02 00 	cmpq   $0x0,0x280(%rsp)
   140793b6e:	00 00 
   140793b70:	74 10                	je     0x140793b82
   140793b72:	48 83 bc 24 a0 02 00 	cmpq   $0x0,0x2a0(%rsp)
   140793b79:	00 00 
   140793b7b:	40 0f 94 c7          	sete   %dil
   140793b7f:	90                   	nop
   140793b80:	eb 06                	jmp    0x140793b88
   140793b82:	31 ff                	xor    %edi,%edi
   140793b84:	eb 02                	jmp    0x140793b88
   140793b86:	31 ff                	xor    %edi,%edi
   140793b88:	40 84 ff             	test   %dil,%dil
   140793b8b:	74 32                	je     0x140793bbf
   140793b8d:	48 8b bc 24 b0 02 00 	mov    0x2b0(%rsp),%rdi
   140793b94:	00 
   140793b95:	48 85 ff             	test   %rdi,%rdi
   140793b98:	7e 25                	jle    0x140793bbf
   140793b9a:	4c 8b 8c 24 e8 02 00 	mov    0x2e8(%rsp),%r9
   140793ba1:	00 
   140793ba2:	49 39 f9             	cmp    %rdi,%r9
   140793ba5:	75 18                	jne    0x140793bbf
   140793ba7:	48 39 bc 24 70 02 00 	cmp    %rdi,0x270(%rsp)
   140793bae:	00 
   140793baf:	75 0e                	jne    0x140793bbf
   140793bb1:	4c 39 8c 24 a8 02 00 	cmp    %r9,0x2a8(%rsp)
   140793bb8:	00 
   140793bb9:	40 0f 95 c7          	setne  %dil
   140793bbd:	eb 02                	jmp    0x140793bc1
   140793bbf:	31 ff                	xor    %edi,%edi
   140793bc1:	83 f7 01             	xor    $0x1,%edi
   140793bc4:	eb 05                	jmp    0x140793bcb
   140793bc6:	bf 01 00 00 00       	mov    $0x1,%edi
   140793bcb:	40 84 ff             	test   %dil,%dil
   140793bce:	0f 85 d9 00 00 00    	jne    0x140793cad
   140793bd4:	48 8b bc 16 80 01 00 	mov    0x180(%rsi,%rdx,1),%rdi
   140793bdb:	00 
   140793bdc:	0f 1f 40 00          	nopl   0x0(%rax)
   140793be0:	4c 39 c7             	cmp    %r8,%rdi
   140793be3:	0f 85 c4 00 00 00    	jne    0x140793cad
   140793be9:	48 89 94 24 48 04 00 	mov    %rdx,0x448(%rsp)
   140793bf0:	00 
   140793bf1:	48 89 b4 24 48 01 00 	mov    %rsi,0x148(%rsp)
   140793bf8:	00 
   140793bf9:	e8 a2 e0 92 ff       	call   0x1400c1ca0 # time.Now
   140793bfe:	48 89 84 24 50 01 00 	mov    %rax,0x150(%rsp)
   140793c05:	00 
   140793c06:	48 89 9c 24 58 01 00 	mov    %rbx,0x158(%rsp)
   140793c0d:	00 
   140793c0e:	48 89 8c 24 60 01 00 	mov    %rcx,0x160(%rsp)
   140793c15:	00 
   140793c16:	90                   	nop
   140793c17:	90                   	nop
   140793c18:	48 0f ba e0 3f       	bt     $0x3f,%rax
   140793c1d:	73 2c                	jae    0x140793c4b
   140793c1f:	48 89 c2             	mov    %rax,%rdx
   140793c22:	25 ff ff ff 3f       	and    $0x3fffffff,%eax
   140793c27:	48 01 d2             	add    %rdx,%rdx
   140793c2a:	48 c1 ea 1f          	shr    $0x1f,%rdx
   140793c2e:	49 b8 80 7f b1 d7 0d 	movabs $0xdd7b17f80,%r8
   140793c35:	00 00 00 
   140793c38:	4c 01 c2             	add    %r8,%rdx
   140793c3b:	48 89 94 24 58 01 00 	mov    %rdx,0x158(%rsp)
   140793c42:	00 
   140793c43:	48 89 84 24 50 01 00 	mov    %rax,0x150(%rsp)
   140793c4a:	00 
   140793c4b:	48 c7 84 24 60 01 00 	movq   $0x0,0x160(%rsp)
   140793c52:	00 00 00 00 00 
   140793c57:	48 8b 84 24 50 01 00 	mov    0x150(%rsp),%rax
   140793c5e:	00 
   140793c5f:	48 8b 9c 24 58 01 00 	mov    0x158(%rsp),%rbx
   140793c66:	00 
   140793c67:	31 c9                	xor    %ecx,%ecx
   140793c69:	48 8d 3d db 02 11 00 	lea    0x1102db(%rip),%rdi        # 0x1408a3f4b
   140793c70:	be 19 00 00 00       	mov    $0x19,%esi
   140793c75:	e8 66 eb 91 ff       	call   0x1400b27e0 # time.Time.Format
   140793c7a:	48 8b 94 24 28 04 00 	mov    0x428(%rsp),%rdx
   140793c81:	00 
   140793c82:	45 31 c0             	xor    %r8d,%r8d
   140793c85:	45 31 c9             	xor    %r9d,%r9d
   140793c88:	4c 8b 94 24 68 04 00 	mov    0x468(%rsp),%r10
   140793c8f:	00 
   140793c90:	4c 8b 9c 24 70 04 00 	mov    0x470(%rsp),%r11
   140793c97:	00 
   140793c98:	4c 8b a4 24 48 04 00 	mov    0x448(%rsp),%r12
   140793c9f:	00 
   140793ca0:	4c 8b ac 24 48 01 00 	mov    0x148(%rsp),%r13
   140793ca7:	00 
   140793ca8:	e9 d3 00 00 00       	jmp    0x140793d80
   140793cad:	90                   	nop
   140793cae:	90                   	nop
   140793caf:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   140793cb4:	48 8b 94 24 60 04 00 	mov    0x460(%rsp),%rdx
   140793cbb:	00 
   140793cbc:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   140793cc3:	00 
   140793cc4:	ff cb                	dec    %ebx
   140793cc6:	74 0d                	je     0x140793cd5
   140793cc8:	48 8b 84 24 38 04 00 	mov    0x438(%rsp),%rax
   140793ccf:	00 
   140793cd0:	e8 8b 4d 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   140793cd5:	b8 10 00 00 00       	mov    $0x10,%eax
   140793cda:	48 8d 1d ff 90 bd 00 	lea    0xbd90ff(%rip),%rbx        # 0x14136cde0
   140793ce1:	b9 01 00 00 00       	mov    $0x1,%ecx
   140793ce6:	e8 15 ea 88 ff       	call   0x140022700 # runtime.mallocgcSmallScanNoHeaderSC2
   140793ceb:	48 c7 40 08 4e 00 00 	movq   $0x4e,0x8(%rax)
   140793cf2:	00 
   140793cf3:	48 8d 15 07 c0 13 00 	lea    0x13c007(%rip),%rdx        # 0x1408cfd01
   140793cfa:	48 89 10             	mov    %rdx,(%rax)
   140793cfd:	48 8d 1d 9c e1 c7 00 	lea    0xc7e19c(%rip),%rbx        # 0x141411ea0
   140793d04:	48 89 c1             	mov    %rax,%rcx
   140793d07:	b8 99 01 00 00       	mov    $0x199,%eax
   140793d0c:	48 81 c4 50 04 00 00 	add    $0x450,%rsp
   140793d13:	5d                   	pop    %rbp
   140793d14:	c3                   	ret
   140793d15:	90                   	nop
   140793d16:	90                   	nop
   140793d17:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   140793d1c:	48 8b 94 24 60 04 00 	mov    0x460(%rsp),%rdx
   140793d23:	00 
   140793d24:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   140793d2b:	00 
   140793d2c:	ff cb                	dec    %ebx
   140793d2e:	74 0d                	je     0x140793d3d
   140793d30:	48 8b 84 24 38 04 00 	mov    0x438(%rsp),%rax
   140793d37:	00 
   140793d38:	e8 23 4d 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   140793d3d:	b8 10 00 00 00       	mov    $0x10,%eax
   140793d42:	48 8d 1d 97 90 bd 00 	lea    0xbd9097(%rip),%rbx        # 0x14136cde0
   140793d49:	b9 01 00 00 00       	mov    $0x1,%ecx
   140793d4e:	e8 ad e9 88 ff       	call   0x140022700 # runtime.mallocgcSmallScanNoHeaderSC2
   140793d53:	48 c7 40 08 20 00 00 	movq   $0x20,0x8(%rax)
   140793d5a:	00 
   140793d5b:	48 8d 15 92 9c 11 00 	lea    0x119c92(%rip),%rdx        # 0x1408ad9f4
   140793d62:	48 89 10             	mov    %rdx,(%rax)
   140793d65:	48 8d 1d 34 e1 c7 00 	lea    0xc7e134(%rip),%rbx        # 0x141411ea0
   140793d6c:	48 89 c1             	mov    %rax,%rcx
   140793d6f:	b8 94 01 00 00       	mov    $0x194,%eax
   140793d74:	48 81 c4 50 04 00 00 	add    $0x450,%rsp
   140793d7b:	5d                   	pop    %rbp
   140793d7c:	c3                   	ret
   140793d7d:	49 ff c1             	inc    %r9
   140793d80:	49 39 d1             	cmp    %rdx,%r9
   140793d83:	0f 8d a2 00 00 00    	jge    0x140793e2b
   140793d89:	48 8b b4 24 28 04 00 	mov    0x428(%rsp),%rsi
   140793d90:	00 
   140793d91:	49 39 f1             	cmp    %rsi,%r9
   140793d94:	0f 83 d6 02 00 00    	jae    0x140794070
   140793d9a:	48 8b b4 24 20 04 00 	mov    0x420(%rsp),%rsi
   140793da1:	00 
   140793da2:	49 69 f9 b8 00 00 00 	imul   $0xb8,%r9,%rdi
   140793da9:	4c 8b 7c 3e 58       	mov    0x58(%rsi,%rdi,1),%r15
   140793dae:	4d 39 d7             	cmp    %r10,%r15
   140793db1:	75 ca                	jne    0x140793d7d
   140793db3:	4c 8b 3c 3e          	mov    (%rsi,%rdi,1),%r15
   140793db7:	4d 39 df             	cmp    %r11,%r15
   140793dba:	75 c1                	jne    0x140793d7d
   140793dbc:	4c 8b 7c 3e 60       	mov    0x60(%rsi,%rdi,1),%r15
   140793dc1:	4b 8b 8c 25 80 01 00 	mov    0x180(%r13,%r12,1),%rcx
   140793dc8:	00 
   140793dc9:	4c 39 f9             	cmp    %r15,%rcx
   140793dcc:	74 0d                	je     0x140793ddb
   140793dce:	4b 8b 8c 2c 40 01 00 	mov    0x140(%r12,%r13,1),%rcx
   140793dd5:	00 
   140793dd6:	4c 39 f9             	cmp    %r15,%rcx
   140793dd9:	75 a2                	jne    0x140793d7d
   140793ddb:	44 0f b6 7c 3e 08    	movzbl 0x8(%rsi,%rdi,1),%r15d
   140793de1:	45 84 ff             	test   %r15b,%r15b
   140793de4:	74 97                	je     0x140793d7d
   140793de6:	c6 44 3e 08 00       	movb   $0x0,0x8(%rsi,%rdi,1)
   140793deb:	48 89 9c 3e a0 00 00 	mov    %rbx,0xa0(%rsi,%rdi,1)
   140793df2:	00 
   140793df3:	83 3d 66 1b 30 04 00 	cmpl   $0x0,0x4301b66(%rip)        # 0x144a95960
   140793dfa:	74 1a                	je     0x140793e16
   140793dfc:	4c 8b bc 3e 98 00 00 	mov    0x98(%rsi,%rdi,1),%r15
   140793e03:	00 
   140793e04:	4c 89 d9             	mov    %r11,%rcx
   140793e07:	e8 d4 9f 8f ff       	call   0x14008dde0 # runtime.gcWriteBarrier2
   140793e0c:	49 89 03             	mov    %rax,(%r11)
   140793e0f:	4d 89 7b 08          	mov    %r15,0x8(%r11)
   140793e13:	49 89 cb             	mov    %rcx,%r11
   140793e16:	48 89 84 3e 98 00 00 	mov    %rax,0x98(%rsi,%rdi,1)
   140793e1d:	00 
   140793e1e:	48 ff 44 3e 40       	incq   0x40(%rsi,%rdi,1)
   140793e23:	49 ff c0             	inc    %r8
   140793e26:	e9 52 ff ff ff       	jmp    0x140793d7d
   140793e2b:	49 83 f8 02          	cmp    $0x2,%r8
   140793e2f:	74 68                	je     0x140793e99
   140793e31:	90                   	nop
   140793e32:	90                   	nop
   140793e33:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   140793e38:	48 8b 94 24 60 04 00 	mov    0x460(%rsp),%rdx
   140793e3f:	00 
   140793e40:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   140793e47:	00 
   140793e48:	ff cb                	dec    %ebx
   140793e4a:	74 0d                	je     0x140793e59
   140793e4c:	48 8b 84 24 38 04 00 	mov    0x438(%rsp),%rax
   140793e53:	00 
   140793e54:	e8 07 4c 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   140793e59:	b8 10 00 00 00       	mov    $0x10,%eax
   140793e5e:	48 8d 1d 7b 8f bd 00 	lea    0xbd8f7b(%rip),%rbx        # 0x14136cde0
   140793e65:	b9 01 00 00 00       	mov    $0x1,%ecx
   140793e6a:	e8 91 e8 88 ff       	call   0x140022700 # runtime.mallocgcSmallScanNoHeaderSC2
   140793e6f:	48 c7 40 08 31 00 00 	movq   $0x31,0x8(%rax)
   140793e76:	00 
   140793e77:	48 8d 15 5c be 12 00 	lea    0x12be5c(%rip),%rdx        # 0x1408bfcda
   140793e7e:	48 89 10             	mov    %rdx,(%rax)
   140793e81:	48 8d 1d 18 e0 c7 00 	lea    0xc7e018(%rip),%rbx        # 0x141411ea0
   140793e88:	48 89 c1             	mov    %rax,%rcx
   140793e8b:	b8 99 01 00 00       	mov    $0x199,%eax
   140793e90:	48 81 c4 50 04 00 00 	add    $0x450,%rsp
   140793e97:	5d                   	pop    %rbp
   140793e98:	c3                   	ret
   140793e99:	4b 89 9c 2c 78 01 00 	mov    %rbx,0x178(%r12,%r13,1)
   140793ea0:	00 
   140793ea1:	83 3d b8 1a 30 04 00 	cmpl   $0x0,0x4301ab8(%rip)        # 0x144a95960
   140793ea8:	74 2c                	je     0x140793ed6
   140793eaa:	4b 8b 8c 2c 70 01 00 	mov    0x170(%r12,%r13,1),%rcx
   140793eb1:	00 
   140793eb2:	4b 8b 94 2c 30 01 00 	mov    0x130(%r12,%r13,1),%rdx
   140793eb9:	00 
   140793eba:	4b 8b b4 2c f8 00 00 	mov    0xf8(%r12,%r13,1),%rsi
   140793ec1:	00 
   140793ec2:	e8 59 9f 8f ff       	call   0x14008de20 # runtime.gcWriteBarrier4
   140793ec7:	49 89 03             	mov    %rax,(%r11)
   140793eca:	49 89 4b 08          	mov    %rcx,0x8(%r11)
   140793ece:	49 89 53 10          	mov    %rdx,0x10(%r11)
   140793ed2:	49 89 73 18          	mov    %rsi,0x18(%r11)
   140793ed6:	4b 89 84 2c 70 01 00 	mov    %rax,0x170(%r12,%r13,1)
   140793edd:	00 
   140793ede:	4b 89 9c 2c 38 01 00 	mov    %rbx,0x138(%r12,%r13,1)
   140793ee5:	00 
   140793ee6:	4b 89 84 2c 30 01 00 	mov    %rax,0x130(%r12,%r13,1)
   140793eed:	00 
   140793eee:	43 c6 84 2c e0 00 00 	movb   $0x0,0xe0(%r12,%r13,1)
   140793ef5:	00 00 
   140793ef7:	4b 89 9c 2c 00 01 00 	mov    %rbx,0x100(%r12,%r13,1)
   140793efe:	00 
   140793eff:	4b 89 84 2c f8 00 00 	mov    %rax,0xf8(%r12,%r13,1)
   140793f06:	00 
   140793f07:	4b ff 84 2c 90 01 00 	incq   0x190(%r12,%r13,1)
   140793f0e:	00 
   140793f0f:	48 89 e1             	mov    %rsp,%rcx
   140793f12:	48 8d 94 24 00 03 00 	lea    0x300(%rsp),%rdx
   140793f19:	00 
   140793f1a:	bb 04 00 00 00       	mov    $0x4,%ebx
   140793f1f:	44 0f 10 32          	movups (%rdx),%xmm14
   140793f23:	44 0f 11 31          	movups %xmm14,(%rcx)
   140793f27:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   140793f2c:	44 0f 11 71 10       	movups %xmm14,0x10(%rcx)
   140793f31:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   140793f36:	44 0f 11 71 20       	movups %xmm14,0x20(%rcx)
   140793f3b:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   140793f40:	44 0f 11 71 30       	movups %xmm14,0x30(%rcx)
   140793f45:	48 83 c2 40          	add    $0x40,%rdx
   140793f49:	48 83 c1 40          	add    $0x40,%rcx
   140793f4d:	ff cb                	dec    %ebx
   140793f4f:	75 ce                	jne    0x140793f1f
   140793f51:	44 0f 10 32          	movups (%rdx),%xmm14
   140793f55:	44 0f 11 31          	movups %xmm14,(%rcx)
   140793f59:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   140793f5e:	44 0f 11 71 10       	movups %xmm14,0x10(%rcx)
   140793f63:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   140793f68:	44 0f 11 71 20       	movups %xmm14,0x20(%rcx)
   140793f6d:	44 0f 10 72 28       	movups 0x28(%rdx),%xmm14
   140793f72:	44 0f 11 71 28       	movups %xmm14,0x28(%rcx)
   140793f77:	48 8b 84 24 60 04 00 	mov    0x460(%rsp),%rax
   140793f7e:	00 
   140793f7f:	90                   	nop
   140793f80:	e8 db 84 ff ff       	call   0x14078c460 # main.(*Host).commit
   140793f85:	48 85 c0             	test   %rax,%rax
   140793f88:	74 68                	je     0x140793ff2
   140793f8a:	90                   	nop
   140793f8b:	90                   	nop
   140793f8c:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   140793f91:	48 8b 94 24 60 04 00 	mov    0x460(%rsp),%rdx
   140793f98:	00 
   140793f99:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   140793fa0:	00 
   140793fa1:	ff cb                	dec    %ebx
   140793fa3:	74 0d                	je     0x140793fb2
   140793fa5:	48 8b 84 24 38 04 00 	mov    0x438(%rsp),%rax
   140793fac:	00 
   140793fad:	e8 ae 4a 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   140793fb2:	b8 10 00 00 00       	mov    $0x10,%eax
   140793fb7:	48 8d 1d 22 8e bd 00 	lea    0xbd8e22(%rip),%rbx        # 0x14136cde0
   140793fbe:	b9 01 00 00 00       	mov    $0x1,%ecx
   140793fc3:	e8 38 e7 88 ff       	call   0x140022700 # runtime.mallocgcSmallScanNoHeaderSC2
   140793fc8:	48 c7 40 08 48 00 00 	movq   $0x48,0x8(%rax)
   140793fcf:	00 
   140793fd0:	48 8d 15 6a 9d 13 00 	lea    0x139d6a(%rip),%rdx        # 0x1408cdd41
   140793fd7:	48 89 10             	mov    %rdx,(%rax)
   140793fda:	48 8d 1d bf de c7 00 	lea    0xc7debf(%rip),%rbx        # 0x141411ea0
   140793fe1:	48 89 c1             	mov    %rax,%rcx
   140793fe4:	b8 f7 01 00 00       	mov    $0x1f7,%eax
   140793fe9:	48 81 c4 50 04 00 00 	add    $0x450,%rsp
   140793ff0:	5d                   	pop    %rbp
   140793ff1:	c3                   	ret
   140793ff2:	90                   	nop
   140793ff3:	90                   	nop
   140793ff4:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   140793ff9:	48 8b b4 24 60 04 00 	mov    0x460(%rsp),%rsi
   140794000:	00 
   140794001:	f0 0f c1 9e d0 00 00 	lock xadd %ebx,0xd0(%rsi)
   140794008:	00 
   140794009:	ff cb                	dec    %ebx
   14079400b:	74 15                	je     0x140794022
   14079400d:	48 8b 84 24 38 04 00 	mov    0x438(%rsp),%rax
   140794014:	00 
   140794015:	e8 46 4a 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   14079401a:	48 8b b4 24 60 04 00 	mov    0x460(%rsp),%rsi
   140794021:	00 
   140794022:	48 8b be c0 00 00 00 	mov    0xc0(%rsi),%rdi
   140794029:	48 85 ff             	test   %rdi,%rdi
   14079402c:	74 1f                	je     0x14079404d
   14079402e:	48 8b 47 78          	mov    0x78(%rdi),%rax
   140794032:	90                   	nop
   140794033:	48 8d 9c 24 40 01 00 	lea    0x140(%rsp),%rbx
   14079403a:	00 
   14079403b:	0f 1f 44 00 00       	nopl   0x0(%rax,%rax,1)
   140794040:	e8 fb 66 88 ff       	call   0x14001a740 # runtime.selectnbsend
   140794045:	48 8b b4 24 60 04 00 	mov    0x460(%rsp),%rsi
   14079404c:	00 
   14079404d:	48 8b 96 c0 02 00 00 	mov    0x2c0(%rsi),%rdx
   140794054:	48 85 d2             	test   %rdx,%rdx
   140794057:	74 05                	je     0x14079405e
   140794059:	48 8b 02             	mov    (%rdx),%rax
   14079405c:	ff d0                	call   *%rax
   14079405e:	b8 c8 00 00 00       	mov    $0xc8,%eax
   140794063:	31 db                	xor    %ebx,%ebx
   140794065:	31 c9                	xor    %ecx,%ecx
   140794067:	48 81 c4 50 04 00 00 	add    $0x450,%rsp
   14079406e:	5d                   	pop    %rbp
   14079406f:	c3                   	ret
   140794070:	e8 0b a1 8f ff       	call   0x14008e180 # runtime.panicBounds
   140794075:	e8 06 a1 8f ff       	call   0x14008e180 # runtime.panicBounds
   14079407a:	90                   	nop
   14079407b:	48 89 44 24 08       	mov    %rax,0x8(%rsp)
   140794080:	48 89 5c 24 10       	mov    %rbx,0x10(%rsp)
   140794085:	48 89 4c 24 18       	mov    %rcx,0x18(%rsp)
   14079408a:	e8 b1 82 8f ff       	call   0x14008c340 # runtime.morestack_noctxt
   14079408f:	48 8b 44 24 08       	mov    0x8(%rsp),%rax
   140794094:	48 8b 5c 24 10       	mov    0x10(%rsp),%rbx
   140794099:	48 8b 4c 24 18       	mov    0x18(%rsp),%rcx
   14079409e:	66 90                	xchg   %ax,%ax
   1407940a0:	e9 7b f6 ff ff       	jmp    0x140793720
   1407940a5:	cc                   	int3
   1407940a6:	cc                   	int3
   1407940a7:	cc                   	int3
   1407940a8:	cc                   	int3
   1407940a9:	cc                   	int3
   1407940aa:	cc                   	int3
   1407940ab:	cc                   	int3
   1407940ac:	cc                   	int3
   1407940ad:	cc                   	int3
   1407940ae:	cc                   	int3
   1407940af:	cc                   	int3
   1407940b0:	cc                   	int3
   1407940b1:	cc                   	int3
   1407940b2:	cc                   	int3
   1407940b3:	cc                   	int3
   1407940b4:	cc                   	int3
   1407940b5:	cc                   	int3
   1407940b6:	cc                   	int3
   1407940b7:	cc                   	int3
   1407940b8:	cc                   	int3
   1407940b9:	cc                   	int3
   1407940ba:	cc                   	int3
   1407940bb:	cc                   	int3
   1407940bc:	cc                   	int3
   1407940bd:	cc                   	int3
   1407940be:	cc                   	int3
   1407940bf:	cc                   	int3

# main.(*Host).approveInspection @ 0x140795a60
   140795a60:	4c 8d a4 24 b8 f9 ff 	lea    -0x648(%rsp),%r12
   140795a67:	ff 
   140795a68:	4d 3b 66 10          	cmp    0x10(%r14),%r12
   140795a6c:	0f 86 23 12 00 00    	jbe    0x140796c95
   140795a72:	55                   	push   %rbp
   140795a73:	48 89 e5             	mov    %rsp,%rbp
   140795a76:	48 81 ec c0 06 00 00 	sub    $0x6c0,%rsp
   140795a7d:	48 89 bc 24 e8 06 00 	mov    %rdi,0x6e8(%rsp)
   140795a84:	00 
   140795a85:	48 8d 94 24 a0 06 00 	lea    0x6a0(%rsp),%rdx
   140795a8c:	00 
   140795a8d:	44 0f 11 3a          	movups %xmm15,(%rdx)
   140795a91:	44 0f 11 7a 10       	movups %xmm15,0x10(%rdx)
   140795a96:	48 83 fe 08          	cmp    $0x8,%rsi
   140795a9a:	75 0f                	jne    0x140795aab
   140795a9c:	48 ba 73 68 69 70 70 	movabs $0x676e697070696873,%rdx
   140795aa3:	69 6e 67 
   140795aa6:	48 39 17             	cmp    %rdx,(%rdi)
   140795aa9:	74 35                	je     0x140795ae0
   140795aab:	48 83 fe 07          	cmp    $0x7,%rsi
   140795aaf:	0f 85 3c 03 00 00    	jne    0x140795df1
   140795ab5:	81 3f 73 65 74 74    	cmpl   $0x74746573,(%rdi)
   140795abb:	0f 1f 44 00 00       	nopl   0x0(%rax,%rax,1)
   140795ac0:	0f 85 2b 03 00 00    	jne    0x140795df1
   140795ac6:	66 81 7f 04 69 6e    	cmpw   $0x6e69,0x4(%rdi)
   140795acc:	0f 85 1f 03 00 00    	jne    0x140795df1
   140795ad2:	80 7f 06 67          	cmpb   $0x67,0x6(%rdi)
   140795ad6:	0f 85 15 03 00 00    	jne    0x140795df1
   140795adc:	0f 1f 40 00          	nopl   0x0(%rax)
   140795ae0:	48 85 db             	test   %rbx,%rbx
   140795ae3:	0f 8e 08 03 00 00    	jle    0x140795df1
   140795ae9:	48 85 c9             	test   %rcx,%rcx
   140795aec:	0f 8e ff 02 00 00    	jle    0x140795df1
   140795af2:	48 89 84 24 d0 06 00 	mov    %rax,0x6d0(%rsp)
   140795af9:	00 
   140795afa:	48 89 9c 24 d8 06 00 	mov    %rbx,0x6d8(%rsp)
   140795b01:	00 
   140795b02:	48 89 8c 24 e0 06 00 	mov    %rcx,0x6e0(%rsp)
   140795b09:	00 
   140795b0a:	48 89 b4 24 f0 06 00 	mov    %rsi,0x6f0(%rsp)
   140795b11:	00 
   140795b12:	48 89 bc 24 e8 06 00 	mov    %rdi,0x6e8(%rsp)
   140795b19:	00 
   140795b1a:	48 8d 90 d0 00 00 00 	lea    0xd0(%rax),%rdx
   140795b21:	48 89 94 24 78 06 00 	mov    %rdx,0x678(%rsp)
   140795b28:	00 
   140795b29:	49 89 c0             	mov    %rax,%r8
   140795b2c:	31 c0                	xor    %eax,%eax
   140795b2e:	41 b9 01 00 00 00    	mov    $0x1,%r9d
   140795b34:	f0 45 0f b1 88 d0 00 	lock cmpxchg %r9d,0xd0(%r8)
   140795b3b:	00 00 
   140795b3d:	41 0f 94 c1          	sete   %r9b
   140795b41:	45 84 c9             	test   %r9b,%r9b
   140795b44:	75 38                	jne    0x140795b7e
   140795b46:	48 89 d0             	mov    %rdx,%rax
   140795b49:	e8 32 2c 90 ff       	call   0x140098780 # internal/sync.(*Mutex).lockSlow
   140795b4e:	48 8b 8c 24 e0 06 00 	mov    0x6e0(%rsp),%rcx
   140795b55:	00 
   140795b56:	48 8b 94 24 78 06 00 	mov    0x678(%rsp),%rdx
   140795b5d:	00 
   140795b5e:	48 8b 9c 24 d8 06 00 	mov    0x6d8(%rsp),%rbx
   140795b65:	00 
   140795b66:	48 8b b4 24 f0 06 00 	mov    0x6f0(%rsp),%rsi
   140795b6d:	00 
   140795b6e:	48 8b bc 24 e8 06 00 	mov    0x6e8(%rsp),%rdi
   140795b75:	00 
   140795b76:	4c 8b 84 24 d0 06 00 	mov    0x6d0(%rsp),%r8
   140795b7d:	00 
   140795b7e:	4d 8d 88 d8 00 00 00 	lea    0xd8(%r8),%r9
   140795b85:	4d 8b 90 e8 01 00 00 	mov    0x1e8(%r8),%r10
   140795b8c:	4d 8b 98 e0 01 00 00 	mov    0x1e0(%r8),%r11
   140795b93:	4c 8d a4 24 a0 03 00 	lea    0x3a0(%rsp),%r12
   140795b9a:	00 
   140795b9b:	41 bd 04 00 00 00    	mov    $0x4,%r13d
   140795ba1:	45 0f 10 31          	movups (%r9),%xmm14
   140795ba5:	45 0f 11 34 24       	movups %xmm14,(%r12)
   140795baa:	45 0f 10 71 10       	movups 0x10(%r9),%xmm14
   140795baf:	45 0f 11 74 24 10    	movups %xmm14,0x10(%r12)
   140795bb5:	45 0f 10 71 20       	movups 0x20(%r9),%xmm14
   140795bba:	45 0f 11 74 24 20    	movups %xmm14,0x20(%r12)
   140795bc0:	45 0f 10 71 30       	movups 0x30(%r9),%xmm14
   140795bc5:	45 0f 11 74 24 30    	movups %xmm14,0x30(%r12)
   140795bcb:	49 83 c1 40          	add    $0x40,%r9
   140795bcf:	49 83 c4 40          	add    $0x40,%r12
   140795bd3:	41 ff cd             	dec    %r13d
   140795bd6:	75 c9                	jne    0x140795ba1
   140795bd8:	45 0f 10 31          	movups (%r9),%xmm14
   140795bdc:	45 0f 11 34 24       	movups %xmm14,(%r12)
   140795be1:	45 0f 10 71 10       	movups 0x10(%r9),%xmm14
   140795be6:	45 0f 11 74 24 10    	movups %xmm14,0x10(%r12)
   140795bec:	45 0f 10 71 20       	movups 0x20(%r9),%xmm14
   140795bf1:	45 0f 11 74 24 20    	movups %xmm14,0x20(%r12)
   140795bf7:	45 0f 10 71 28       	movups 0x28(%r9),%xmm14
   140795bfc:	45 0f 11 74 24 28    	movups %xmm14,0x28(%r12)
   140795c02:	4d 85 d2             	test   %r10,%r10
   140795c05:	75 0a                	jne    0x140795c11
   140795c07:	31 c0                	xor    %eax,%eax
   140795c09:	45 31 c9             	xor    %r9d,%r9d
   140795c0c:	45 31 e4             	xor    %r12d,%r12d
   140795c0f:	eb 3f                	jmp    0x140795c50
   140795c11:	4c 89 94 24 60 01 00 	mov    %r10,0x160(%rsp)
   140795c18:	00 
   140795c19:	4c 89 9c 24 98 06 00 	mov    %r11,0x698(%rsp)
   140795c20:	00 
   140795c21:	31 c0                	xor    %eax,%eax
   140795c23:	4c 89 d3             	mov    %r10,%rbx
   140795c26:	31 c9                	xor    %ecx,%ecx
   140795c28:	48 89 df             	mov    %rbx,%rdi
   140795c2b:	48 8d 35 fe fa c6 00 	lea    0xc6fafe(%rip),%rsi        # 0x141405730
   140795c32:	e8 c9 30 8f ff       	call   0x140088d00 # runtime.growslice
   140795c37:	4c 8b 94 24 60 01 00 	mov    0x160(%rsp),%r10
   140795c3e:	00 
   140795c3f:	4c 8b 9c 24 98 06 00 	mov    0x698(%rsp),%r11
   140795c46:	00 
   140795c47:	49 89 c1             	mov    %rax,%r9
   140795c4a:	49 89 cc             	mov    %rcx,%r12
   140795c4d:	48 89 d8             	mov    %rbx,%rax
   140795c50:	4c 89 a4 24 60 01 00 	mov    %r12,0x160(%rsp)
   140795c57:	00 
   140795c58:	48 89 84 24 58 01 00 	mov    %rax,0x158(%rsp)
   140795c5f:	00 
   140795c60:	4c 89 8c 24 98 06 00 	mov    %r9,0x698(%rsp)
   140795c67:	00 
   140795c68:	4c 89 cb             	mov    %r9,%rbx
   140795c6b:	48 89 c1             	mov    %rax,%rcx
   140795c6e:	4c 89 df             	mov    %r11,%rdi
   140795c71:	4c 89 d6             	mov    %r10,%rsi
   140795c74:	48 8d 05 b5 fa c6 00 	lea    0xc6fab5(%rip),%rax        # 0x141405730
   140795c7b:	0f 1f 44 00 00       	nopl   0x0(%rax,%rax,1)
   140795c80:	e8 5b f9 8e ff       	call   0x1400855e0 # runtime.typedslicecopy
   140795c85:	48 8b 94 24 98 06 00 	mov    0x698(%rsp),%rdx
   140795c8c:	00 
   140795c8d:	48 89 94 24 a8 04 00 	mov    %rdx,0x4a8(%rsp)
   140795c94:	00 
   140795c95:	48 8b 94 24 58 01 00 	mov    0x158(%rsp),%rdx
   140795c9c:	00 
   140795c9d:	48 89 94 24 b0 04 00 	mov    %rdx,0x4b0(%rsp)
   140795ca4:	00 
   140795ca5:	48 8b 94 24 60 01 00 	mov    0x160(%rsp),%rdx
   140795cac:	00 
   140795cad:	48 89 94 24 b8 04 00 	mov    %rdx,0x4b8(%rsp)
   140795cb4:	00 
   140795cb5:	48 8b bc 24 c8 04 00 	mov    0x4c8(%rsp),%rdi
   140795cbc:	00 
   140795cbd:	48 8b 94 24 c0 04 00 	mov    0x4c0(%rsp),%rdx
   140795cc4:	00 
   140795cc5:	48 85 ff             	test   %rdi,%rdi
   140795cc8:	75 08                	jne    0x140795cd2
   140795cca:	31 db                	xor    %ebx,%ebx
   140795ccc:	31 c0                	xor    %eax,%eax
   140795cce:	31 c9                	xor    %ecx,%ecx
   140795cd0:	eb 33                	jmp    0x140795d05
   140795cd2:	48 89 bc 24 60 01 00 	mov    %rdi,0x160(%rsp)
   140795cd9:	00 
   140795cda:	48 89 94 24 90 06 00 	mov    %rdx,0x690(%rsp)
   140795ce1:	00 
   140795ce2:	31 c0                	xor    %eax,%eax
   140795ce4:	48 89 fb             	mov    %rdi,%rbx
   140795ce7:	31 c9                	xor    %ecx,%ecx
   140795ce9:	48 8d 35 68 46 c2 00 	lea    0xc24668(%rip),%rsi        # 0x1413ba358
   140795cf0:	e8 0b 30 8f ff       	call   0x140088d00 # runtime.growslice
   140795cf5:	48 8b 94 24 90 06 00 	mov    0x690(%rsp),%rdx
   140795cfc:	00 
   140795cfd:	48 8b bc 24 60 01 00 	mov    0x160(%rsp),%rdi
   140795d04:	00 
   140795d05:	48 89 8c 24 60 01 00 	mov    %rcx,0x160(%rsp)
   140795d0c:	00 
   140795d0d:	48 89 9c 24 58 01 00 	mov    %rbx,0x158(%rsp)
   140795d14:	00 
   140795d15:	48 89 84 24 90 06 00 	mov    %rax,0x690(%rsp)
   140795d1c:	00 
   140795d1d:	48 89 d9             	mov    %rbx,%rcx
   140795d20:	48 89 fe             	mov    %rdi,%rsi
   140795d23:	48 89 c3             	mov    %rax,%rbx
   140795d26:	48 89 d7             	mov    %rdx,%rdi
   140795d29:	48 8d 05 28 46 c2 00 	lea    0xc24628(%rip),%rax        # 0x1413ba358
   140795d30:	e8 ab f8 8e ff       	call   0x1400855e0 # runtime.typedslicecopy
   140795d35:	48 8b 94 24 90 06 00 	mov    0x690(%rsp),%rdx
   140795d3c:	00 
   140795d3d:	48 89 94 24 c0 04 00 	mov    %rdx,0x4c0(%rsp)
   140795d44:	00 
   140795d45:	48 8b 94 24 58 01 00 	mov    0x158(%rsp),%rdx
   140795d4c:	00 
   140795d4d:	48 89 94 24 c8 04 00 	mov    %rdx,0x4c8(%rsp)
   140795d54:	00 
   140795d55:	48 8b 94 24 60 01 00 	mov    0x160(%rsp),%rdx
   140795d5c:	00 
   140795d5d:	48 89 94 24 d0 04 00 	mov    %rdx,0x4d0(%rsp)
   140795d64:	00 
   140795d65:	48 8b 94 24 a8 04 00 	mov    0x4a8(%rsp),%rdx
   140795d6c:	00 
   140795d6d:	4c 8b 84 24 b0 04 00 	mov    0x4b0(%rsp),%r8
   140795d74:	00 
   140795d75:	4c 8d 8c 24 40 05 00 	lea    0x540(%rsp),%r9
   140795d7c:	00 
   140795d7d:	4c 8d 94 24 a0 03 00 	lea    0x3a0(%rsp),%r10
   140795d84:	00 
   140795d85:	41 bb 04 00 00 00    	mov    $0x4,%r11d
   140795d8b:	45 0f 10 32          	movups (%r10),%xmm14
   140795d8f:	45 0f 11 31          	movups %xmm14,(%r9)
   140795d93:	45 0f 10 72 10       	movups 0x10(%r10),%xmm14
   140795d98:	45 0f 11 71 10       	movups %xmm14,0x10(%r9)
   140795d9d:	45 0f 10 72 20       	movups 0x20(%r10),%xmm14
   140795da2:	45 0f 11 71 20       	movups %xmm14,0x20(%r9)
   140795da7:	45 0f 10 72 30       	movups 0x30(%r10),%xmm14
   140795dac:	45 0f 11 71 30       	movups %xmm14,0x30(%r9)
   140795db1:	49 83 c2 40          	add    $0x40,%r10
   140795db5:	49 83 c1 40          	add    $0x40,%r9
   140795db9:	41 ff cb             	dec    %r11d
   140795dbc:	75 cd                	jne    0x140795d8b
   140795dbe:	45 0f 10 32          	movups (%r10),%xmm14
   140795dc2:	45 0f 11 31          	movups %xmm14,(%r9)
   140795dc6:	45 0f 10 72 10       	movups 0x10(%r10),%xmm14
   140795dcb:	45 0f 11 71 10       	movups %xmm14,0x10(%r9)
   140795dd0:	45 0f 10 72 20       	movups 0x20(%r10),%xmm14
   140795dd5:	45 0f 11 71 20       	movups %xmm14,0x20(%r9)
   140795dda:	45 0f 10 72 28       	movups 0x28(%r10),%xmm14
   140795ddf:	45 0f 11 71 28       	movups %xmm14,0x28(%r9)
   140795de4:	45 31 c9             	xor    %r9d,%r9d
   140795de7:	4c 8b 94 24 d8 06 00 	mov    0x6d8(%rsp),%r10
   140795dee:	00 
   140795def:	eb 4f                	jmp    0x140795e40
   140795df1:	b8 10 00 00 00       	mov    $0x10,%eax
   140795df6:	48 8d 1d e3 6f bd 00 	lea    0xbd6fe3(%rip),%rbx        # 0x14136cde0
   140795dfd:	b9 01 00 00 00       	mov    $0x1,%ecx
   140795e02:	e8 f9 c8 88 ff       	call   0x140022700 # runtime.mallocgcSmallScanNoHeaderSC2
   140795e07:	48 c7 40 08 27 00 00 	movq   $0x27,0x8(%rax)
   140795e0e:	00 
   140795e0f:	48 8d 15 89 ff 11 00 	lea    0x11ff89(%rip),%rdx        # 0x1408b5d9f
   140795e16:	48 89 10             	mov    %rdx,(%rax)
   140795e19:	48 8d 1d 80 c0 c7 00 	lea    0xc7c080(%rip),%rbx        # 0x141411ea0
   140795e20:	48 89 c1             	mov    %rax,%rcx
   140795e23:	b8 90 01 00 00       	mov    $0x190,%eax
   140795e28:	48 81 c4 c0 06 00 00 	add    $0x6c0,%rsp
   140795e2f:	5d                   	pop    %rbp
   140795e30:	c3                   	ret
   140795e31:	48 81 c2 98 01 00 00 	add    $0x198,%rdx
   140795e38:	49 ff c1             	inc    %r9
   140795e3b:	0f 1f 44 00 00       	nopl   0x0(%rax,%rax,1)
   140795e40:	4d 39 c1             	cmp    %r8,%r9
   140795e43:	7d 0b                	jge    0x140795e50
   140795e45:	4c 39 92 b8 00 00 00 	cmp    %r10,0xb8(%rdx)
   140795e4c:	75 e3                	jne    0x140795e31
   140795e4e:	eb 07                	jmp    0x140795e57
   140795e50:	49 c7 c1 ff ff ff ff 	mov    $0xffffffffffffffff,%r9
   140795e57:	4c 89 8c 24 40 01 00 	mov    %r9,0x140(%rsp)
   140795e5e:	00 
   140795e5f:	48 8b 94 24 60 06 00 	mov    0x660(%rsp),%rdx
   140795e66:	00 
   140795e67:	48 8b b4 24 68 06 00 	mov    0x668(%rsp),%rsi
   140795e6e:	00 
   140795e6f:	48 89 b4 24 60 01 00 	mov    %rsi,0x160(%rsp)
   140795e76:	00 
   140795e77:	31 ff                	xor    %edi,%edi
   140795e79:	4c 8b 84 24 e0 06 00 	mov    0x6e0(%rsp),%r8
   140795e80:	00 
   140795e81:	4c 8b 9c 24 f0 06 00 	mov    0x6f0(%rsp),%r11
   140795e88:	00 
   140795e89:	eb 0a                	jmp    0x140795e95
   140795e8b:	48 81 c2 b8 00 00 00 	add    $0xb8,%rdx
   140795e92:	48 ff c7             	inc    %rdi
   140795e95:	48 39 f7             	cmp    %rsi,%rdi
   140795e98:	0f 8d 8c 00 00 00    	jge    0x140795f2a
   140795e9e:	66 90                	xchg   %ax,%ax
   140795ea0:	4c 39 42 60          	cmp    %r8,0x60(%rdx)
   140795ea4:	75 75                	jne    0x140795f1b
   140795ea6:	4c 8b 62 58          	mov    0x58(%rdx),%r12
   140795eaa:	4d 39 e2             	cmp    %r12,%r10
   140795ead:	75 68                	jne    0x140795f17
   140795eaf:	48 8b 4a 50          	mov    0x50(%rdx),%rcx
   140795eb3:	49 39 cb             	cmp    %rcx,%r11
   140795eb6:	74 04                	je     0x140795ebc
   140795eb8:	31 c0                	xor    %eax,%eax
   140795eba:	eb 64                	jmp    0x140795f20
   140795ebc:	48 89 bc 24 48 01 00 	mov    %rdi,0x148(%rsp)
   140795ec3:	00 
   140795ec4:	48 89 94 24 88 06 00 	mov    %rdx,0x688(%rsp)
   140795ecb:	00 
   140795ecc:	48 8b 42 48          	mov    0x48(%rdx),%rax
   140795ed0:	48 8b 9c 24 e8 06 00 	mov    0x6e8(%rsp),%rbx
   140795ed7:	00 
   140795ed8:	e8 a3 87 87 ff       	call   0x14000e680 # runtime.memequal
   140795edd:	48 8b 94 24 88 06 00 	mov    0x688(%rsp),%rdx
   140795ee4:	00 
   140795ee5:	48 8b b4 24 60 01 00 	mov    0x160(%rsp),%rsi
   140795eec:	00 
   140795eed:	48 8b bc 24 48 01 00 	mov    0x148(%rsp),%rdi
   140795ef4:	00 
   140795ef5:	4c 8b 84 24 e0 06 00 	mov    0x6e0(%rsp),%r8
   140795efc:	00 
   140795efd:	4c 8b 8c 24 40 01 00 	mov    0x140(%rsp),%r9
   140795f04:	00 
   140795f05:	4c 8b 94 24 d8 06 00 	mov    0x6d8(%rsp),%r10
   140795f0c:	00 
   140795f0d:	4c 8b 9c 24 f0 06 00 	mov    0x6f0(%rsp),%r11
   140795f14:	00 
   140795f15:	eb 09                	jmp    0x140795f20
   140795f17:	31 c0                	xor    %eax,%eax
   140795f19:	eb 05                	jmp    0x140795f20
   140795f1b:	31 c0                	xor    %eax,%eax
   140795f1d:	0f 1f 00             	nopl   (%rax)
   140795f20:	84 c0                	test   %al,%al
   140795f22:	0f 84 63 ff ff ff    	je     0x140795e8b
   140795f28:	eb 07                	jmp    0x140795f31
   140795f2a:	48 c7 c7 ff ff ff ff 	mov    $0xffffffffffffffff,%rdi
   140795f31:	4d 85 c9             	test   %r9,%r9
   140795f34:	0f 8c dc 05 00 00    	jl     0x140796516
   140795f3a:	66 0f 1f 44 00 00    	nopw   0x0(%rax,%rax,1)
   140795f40:	48 85 ff             	test   %rdi,%rdi
   140795f43:	0f 8c cd 05 00 00    	jl     0x140796516
   140795f49:	48 8b 94 24 50 06 00 	mov    0x650(%rsp),%rdx
   140795f50:	00 
   140795f51:	49 39 d1             	cmp    %rdx,%r9
   140795f54:	0f 83 35 0d 00 00    	jae    0x140796c8f
   140795f5a:	48 8b 94 24 48 06 00 	mov    0x648(%rsp),%rdx
   140795f61:	00 
   140795f62:	49 69 f1 98 01 00 00 	imul   $0x198,%r9,%rsi
   140795f69:	48 01 f2             	add    %rsi,%rdx
   140795f6c:	4c 8d 94 24 a0 03 00 	lea    0x3a0(%rsp),%r10
   140795f73:	00 
   140795f74:	41 bc 06 00 00 00    	mov    $0x6,%r12d
   140795f7a:	44 0f 10 32          	movups (%rdx),%xmm14
   140795f7e:	45 0f 11 32          	movups %xmm14,(%r10)
   140795f82:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   140795f87:	45 0f 11 72 10       	movups %xmm14,0x10(%r10)
   140795f8c:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   140795f91:	45 0f 11 72 20       	movups %xmm14,0x20(%r10)
   140795f96:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   140795f9b:	45 0f 11 72 30       	movups %xmm14,0x30(%r10)
   140795fa0:	48 83 c2 40          	add    $0x40,%rdx
   140795fa4:	49 83 c2 40          	add    $0x40,%r10
   140795fa8:	41 ff cc             	dec    %r12d
   140795fab:	75 cd                	jne    0x140795f7a
   140795fad:	44 0f 10 32          	movups (%rdx),%xmm14
   140795fb1:	45 0f 11 32          	movups %xmm14,(%r10)
   140795fb5:	44 0f 10 72 08       	movups 0x8(%rdx),%xmm14
   140795fba:	45 0f 11 72 08       	movups %xmm14,0x8(%r10)
   140795fbf:	48 8d 94 24 08 02 00 	lea    0x208(%rsp),%rdx
   140795fc6:	00 
   140795fc7:	4c 8d 94 24 a0 03 00 	lea    0x3a0(%rsp),%r10
   140795fce:	00 
   140795fcf:	41 bc 06 00 00 00    	mov    $0x6,%r12d
   140795fd5:	45 0f 10 32          	movups (%r10),%xmm14
   140795fd9:	44 0f 11 32          	movups %xmm14,(%rdx)
   140795fdd:	45 0f 10 72 10       	movups 0x10(%r10),%xmm14
   140795fe2:	44 0f 11 72 10       	movups %xmm14,0x10(%rdx)
   140795fe7:	45 0f 10 72 20       	movups 0x20(%r10),%xmm14
   140795fec:	44 0f 11 72 20       	movups %xmm14,0x20(%rdx)
   140795ff1:	45 0f 10 72 30       	movups 0x30(%r10),%xmm14
   140795ff6:	44 0f 11 72 30       	movups %xmm14,0x30(%rdx)
   140795ffb:	49 83 c2 40          	add    $0x40,%r10
   140795fff:	48 83 c2 40          	add    $0x40,%rdx
   140796003:	41 ff cc             	dec    %r12d
   140796006:	75 cd                	jne    0x140795fd5
   140796008:	45 0f 10 32          	movups (%r10),%xmm14
   14079600c:	44 0f 11 32          	movups %xmm14,(%rdx)
   140796010:	45 0f 10 72 08       	movups 0x8(%r10),%xmm14
   140796015:	44 0f 11 72 08       	movups %xmm14,0x8(%rdx)
   14079601a:	48 8d 94 24 70 01 00 	lea    0x170(%rsp),%rdx
   140796021:	00 
   140796022:	44 0f 11 3a          	movups %xmm15,(%rdx)
   140796026:	44 0f 11 7a 10       	movups %xmm15,0x10(%rdx)
   14079602b:	44 0f 11 7a 20       	movups %xmm15,0x20(%rdx)
   140796030:	44 0f 11 7a 30       	movups %xmm15,0x30(%rdx)
   140796035:	49 83 fb 08          	cmp    $0x8,%r11
   140796039:	75 49                	jne    0x140796084
   14079603b:	49 ba 73 68 69 70 70 	movabs $0x676e697070696873,%r10
   140796042:	69 6e 67 
   140796045:	4c 8b a4 24 e8 06 00 	mov    0x6e8(%rsp),%r12
   14079604c:	00 
   14079604d:	4d 39 14 24          	cmp    %r10,(%r12)
   140796051:	75 43                	jne    0x140796096
   140796053:	4c 8d ac 24 10 03 00 	lea    0x310(%rsp),%r13
   14079605a:	00 
   14079605b:	45 0f 10 75 00       	movups 0x0(%r13),%xmm14
   140796060:	44 0f 11 32          	movups %xmm14,(%rdx)
   140796064:	45 0f 10 75 10       	movups 0x10(%r13),%xmm14
   140796069:	44 0f 11 72 10       	movups %xmm14,0x10(%rdx)
   14079606e:	45 0f 10 75 20       	movups 0x20(%r13),%xmm14
   140796073:	44 0f 11 72 20       	movups %xmm14,0x20(%rdx)
   140796078:	45 0f 10 75 30       	movups 0x30(%r13),%xmm14
   14079607d:	44 0f 11 72 30       	movups %xmm14,0x30(%rdx)
   140796082:	eb 41                	jmp    0x1407960c5
   140796084:	49 ba 73 68 69 70 70 	movabs $0x676e697070696873,%r10
   14079608b:	69 6e 67 
   14079608e:	4c 8b a4 24 e8 06 00 	mov    0x6e8(%rsp),%r12
   140796095:	00 
   140796096:	4c 8d ac 24 50 03 00 	lea    0x350(%rsp),%r13
   14079609d:	00 
   14079609e:	45 0f 10 75 00       	movups 0x0(%r13),%xmm14
   1407960a3:	44 0f 11 32          	movups %xmm14,(%rdx)
   1407960a7:	45 0f 10 75 10       	movups 0x10(%r13),%xmm14
   1407960ac:	44 0f 11 72 10       	movups %xmm14,0x10(%rdx)
   1407960b1:	45 0f 10 75 20       	movups 0x20(%r13),%xmm14
   1407960b6:	44 0f 11 72 20       	movups %xmm14,0x20(%rdx)
   1407960bb:	45 0f 10 75 30       	movups 0x30(%r13),%xmm14
   1407960c0:	44 0f 11 72 30       	movups %xmm14,0x30(%rdx)
   1407960c5:	4c 8d ac 24 c8 01 00 	lea    0x1c8(%rsp),%r13
   1407960cc:	00 
   1407960cd:	44 0f 10 32          	movups (%rdx),%xmm14
   1407960d1:	45 0f 11 75 00       	movups %xmm14,0x0(%r13)
   1407960d6:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   1407960db:	45 0f 11 75 10       	movups %xmm14,0x10(%r13)
   1407960e0:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   1407960e5:	45 0f 11 75 20       	movups %xmm14,0x20(%r13)
   1407960ea:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   1407960ef:	45 0f 11 75 30       	movups %xmm14,0x30(%r13)
   1407960f4:	48 8b 94 24 68 06 00 	mov    0x668(%rsp),%rdx
   1407960fb:	00 
   1407960fc:	0f 1f 40 00          	nopl   0x0(%rax)
   140796100:	48 39 d7             	cmp    %rdx,%rdi
   140796103:	0f 83 81 0b 00 00    	jae    0x140796c8a
   140796109:	48 8b 94 24 60 06 00 	mov    0x660(%rsp),%rdx
   140796110:	00 
   140796111:	4c 69 ef b8 00 00 00 	imul   $0xb8,%rdi,%r13
   140796118:	4e 8d 3c 2a          	lea    (%rdx,%r13,1),%r15
   14079611c:	42 0f b6 54 2a 08    	movzbl 0x8(%rdx,%r13,1),%edx
   140796122:	4c 89 e8             	mov    %r13,%rax
   140796125:	4c 8d ac 24 08 02 00 	lea    0x208(%rsp),%r13
   14079612c:	00 
   14079612d:	45 0f 10 37          	movups (%r15),%xmm14
   140796131:	45 0f 11 75 00       	movups %xmm14,0x0(%r13)
   140796136:	45 0f 10 77 10       	movups 0x10(%r15),%xmm14
   14079613b:	45 0f 11 75 10       	movups %xmm14,0x10(%r13)
   140796140:	45 0f 10 77 20       	movups 0x20(%r15),%xmm14
   140796145:	45 0f 11 75 20       	movups %xmm14,0x20(%r13)
   14079614a:	45 0f 10 77 30       	movups 0x30(%r15),%xmm14
   14079614f:	45 0f 11 75 30       	movups %xmm14,0x30(%r13)
   140796154:	45 0f 10 77 40       	movups 0x40(%r15),%xmm14
   140796159:	45 0f 11 75 40       	movups %xmm14,0x40(%r13)
   14079615e:	45 0f 10 77 50       	movups 0x50(%r15),%xmm14
   140796163:	45 0f 11 75 50       	movups %xmm14,0x50(%r13)
   140796168:	45 0f 10 77 60       	movups 0x60(%r15),%xmm14
   14079616d:	45 0f 11 75 60       	movups %xmm14,0x60(%r13)
   140796172:	45 0f 10 77 70       	movups 0x70(%r15),%xmm14
   140796177:	45 0f 11 75 70       	movups %xmm14,0x70(%r13)
   14079617c:	45 0f 10 b7 80 00 00 	movups 0x80(%r15),%xmm14
   140796183:	00 
   140796184:	45 0f 11 b5 80 00 00 	movups %xmm14,0x80(%r13)
   14079618b:	00 
   14079618c:	45 0f 10 b7 90 00 00 	movups 0x90(%r15),%xmm14
   140796193:	00 
   140796194:	45 0f 11 b5 90 00 00 	movups %xmm14,0x90(%r13)
   14079619b:	00 
   14079619c:	45 0f 10 b7 a0 00 00 	movups 0xa0(%r15),%xmm14
   1407961a3:	00 
   1407961a4:	45 0f 11 b5 a0 00 00 	movups %xmm14,0xa0(%r13)
   1407961ab:	00 
   1407961ac:	45 0f 10 b7 a8 00 00 	movups 0xa8(%r15),%xmm14
   1407961b3:	00 
   1407961b4:	45 0f 11 b5 a8 00 00 	movups %xmm14,0xa8(%r13)
   1407961bb:	00 
   1407961bc:	4c 8b ac 24 00 02 00 	mov    0x200(%rsp),%r13
   1407961c3:	00 
   1407961c4:	4d 39 c5             	cmp    %r8,%r13
   1407961c7:	75 4e                	jne    0x140796217
   1407961c9:	48 83 bc 24 f8 01 00 	cmpq   $0x0,0x1f8(%rsp)
   1407961d0:	00 00 
   1407961d2:	74 43                	je     0x140796217
   1407961d4:	84 d2                	test   %dl,%dl
   1407961d6:	75 3f                	jne    0x140796217
   1407961d8:	90                   	nop
   1407961d9:	90                   	nop
   1407961da:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   1407961df:	48 8b 94 24 d0 06 00 	mov    0x6d0(%rsp),%rdx
   1407961e6:	00 
   1407961e7:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   1407961ee:	00 
   1407961ef:	ff cb                	dec    %ebx
   1407961f1:	74 12                	je     0x140796205
   1407961f3:	48 8b 84 24 78 06 00 	mov    0x678(%rsp),%rax
   1407961fa:	00 
   1407961fb:	0f 1f 44 00 00       	nopl   0x0(%rax,%rax,1)
   140796200:	e8 5b 28 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   140796205:	b8 c8 00 00 00       	mov    $0xc8,%eax
   14079620a:	31 db                	xor    %ebx,%ebx
   14079620c:	31 c9                	xor    %ecx,%ecx
   14079620e:	48 81 c4 c0 06 00 00 	add    $0x6c0,%rsp
   140796215:	5d                   	pop    %rbp
   140796216:	c3                   	ret
   140796217:	80 bc 24 80 04 00 00 	cmpb   $0x0,0x480(%rsp)
   14079621e:	00 
   14079621f:	90                   	nop
   140796220:	74 20                	je     0x140796242
   140796222:	48 83 bc 24 d8 01 00 	cmpq   $0x0,0x1d8(%rsp)
   140796229:	00 00 
   14079622b:	74 0e                	je     0x14079623b
   14079622d:	48 83 bc 24 f8 01 00 	cmpq   $0x0,0x1f8(%rsp)
   140796234:	00 00 
   140796236:	0f 94 c2             	sete   %dl
   140796239:	eb 02                	jmp    0x14079623d
   14079623b:	31 d2                	xor    %edx,%edx
   14079623d:	83 f2 01             	xor    $0x1,%edx
   140796240:	eb 05                	jmp    0x140796247
   140796242:	ba 01 00 00 00       	mov    $0x1,%edx
   140796247:	84 d2                	test   %dl,%dl
   140796249:	0f 85 5f 02 00 00    	jne    0x1407964ae
   14079624f:	4d 39 c5             	cmp    %r8,%r13
   140796252:	0f 85 56 02 00 00    	jne    0x1407964ae
   140796258:	80 bc 24 10 02 00 00 	cmpb   $0x0,0x210(%rsp)
   14079625f:	00 
   140796260:	0f 84 48 02 00 00    	je     0x1407964ae
   140796266:	48 89 b4 24 60 01 00 	mov    %rsi,0x160(%rsp)
   14079626d:	00 
   14079626e:	48 89 84 24 58 01 00 	mov    %rax,0x158(%rsp)
   140796275:	00 
   140796276:	48 89 bc 24 48 01 00 	mov    %rdi,0x148(%rsp)
   14079627d:	00 
   14079627e:	66 90                	xchg   %ax,%ax
   140796280:	49 83 fb 08          	cmp    $0x8,%r11
   140796284:	75 15                	jne    0x14079629b
   140796286:	4d 39 14 24          	cmp    %r10,(%r12)
   14079628a:	75 0f                	jne    0x14079629b
   14079628c:	48 83 bc 24 18 05 00 	cmpq   $0x0,0x518(%rsp)
   140796293:	00 00 
   140796295:	0f 84 13 02 00 00    	je     0x1407964ae
   14079629b:	0f 1f 44 00 00       	nopl   0x0(%rax,%rax,1)
   1407962a0:	e8 fb b9 92 ff       	call   0x1400c1ca0 # time.Now
   1407962a5:	48 89 84 24 b0 01 00 	mov    %rax,0x1b0(%rsp)
   1407962ac:	00 
   1407962ad:	48 89 9c 24 b8 01 00 	mov    %rbx,0x1b8(%rsp)
   1407962b4:	00 
   1407962b5:	48 89 8c 24 c0 01 00 	mov    %rcx,0x1c0(%rsp)
   1407962bc:	00 
   1407962bd:	90                   	nop
   1407962be:	90                   	nop
   1407962bf:	48 0f ba e0 3f       	bt     $0x3f,%rax
   1407962c4:	73 2c                	jae    0x1407962f2
   1407962c6:	48 89 c2             	mov    %rax,%rdx
   1407962c9:	25 ff ff ff 3f       	and    $0x3fffffff,%eax
   1407962ce:	48 01 d2             	add    %rdx,%rdx
   1407962d1:	48 c1 ea 1f          	shr    $0x1f,%rdx
   1407962d5:	49 b8 80 7f b1 d7 0d 	movabs $0xdd7b17f80,%r8
   1407962dc:	00 00 00 
   1407962df:	4c 01 c2             	add    %r8,%rdx
   1407962e2:	48 89 94 24 b8 01 00 	mov    %rdx,0x1b8(%rsp)
   1407962e9:	00 
   1407962ea:	48 89 84 24 b0 01 00 	mov    %rax,0x1b0(%rsp)
   1407962f1:	00 
   1407962f2:	48 c7 84 24 c0 01 00 	movq   $0x0,0x1c0(%rsp)
   1407962f9:	00 00 00 00 00 
   1407962fe:	48 8b 84 24 b0 01 00 	mov    0x1b0(%rsp),%rax
   140796305:	00 
   140796306:	48 8b 9c 24 b8 01 00 	mov    0x1b8(%rsp),%rbx
   14079630d:	00 
   14079630e:	31 c9                	xor    %ecx,%ecx
   140796310:	48 8d 3d 34 dc 10 00 	lea    0x10dc34(%rip),%rdi        # 0x1408a3f4b
   140796317:	be 19 00 00 00       	mov    $0x19,%esi
   14079631c:	0f 1f 40 00          	nopl   0x0(%rax)
   140796320:	e8 bb c4 91 ff       	call   0x1400b27e0 # time.Time.Format
   140796325:	48 8b 94 24 f0 06 00 	mov    0x6f0(%rsp),%rdx
   14079632c:	00 
   14079632d:	48 83 fa 07          	cmp    $0x7,%rdx
   140796331:	0f 85 15 01 00 00    	jne    0x14079644c
   140796337:	4c 8b 94 24 e8 06 00 	mov    0x6e8(%rsp),%r10
   14079633e:	00 
   14079633f:	41 81 3a 73 65 74 74 	cmpl   $0x74746573,(%r10)
   140796346:	0f 85 fb 00 00 00    	jne    0x140796447
   14079634c:	66 41 81 7a 04 69 6e 	cmpw   $0x6e69,0x4(%r10)
   140796353:	0f 85 e9 00 00 00    	jne    0x140796442
   140796359:	41 80 7a 06 67       	cmpb   $0x67,0x6(%r10)
   14079635e:	66 90                	xchg   %ax,%ax
   140796360:	0f 85 d7 00 00 00    	jne    0x14079643d
   140796366:	4c 8d 9c 24 08 02 00 	lea    0x208(%rsp),%r11
   14079636d:	00 
   14079636e:	4c 8d a4 24 a0 03 00 	lea    0x3a0(%rsp),%r12
   140796375:	00 
   140796376:	41 bd 06 00 00 00    	mov    $0x6,%r13d
   14079637c:	45 0f 10 34 24       	movups (%r12),%xmm14
   140796381:	45 0f 11 33          	movups %xmm14,(%r11)
   140796385:	45 0f 10 74 24 10    	movups 0x10(%r12),%xmm14
   14079638b:	45 0f 11 73 10       	movups %xmm14,0x10(%r11)
   140796390:	45 0f 10 74 24 20    	movups 0x20(%r12),%xmm14
   140796396:	45 0f 11 73 20       	movups %xmm14,0x20(%r11)
   14079639b:	45 0f 10 74 24 30    	movups 0x30(%r12),%xmm14
   1407963a1:	45 0f 11 73 30       	movups %xmm14,0x30(%r11)
   1407963a6:	49 83 c4 40          	add    $0x40,%r12
   1407963aa:	49 83 c3 40          	add    $0x40,%r11
   1407963ae:	41 ff cd             	dec    %r13d
   1407963b1:	75 c9                	jne    0x14079637c
   1407963b3:	45 0f 10 34 24       	movups (%r12),%xmm14
   1407963b8:	45 0f 11 33          	movups %xmm14,(%r11)
   1407963bc:	45 0f 10 74 24 08    	movups 0x8(%r12),%xmm14
   1407963c2:	45 0f 11 73 08       	movups %xmm14,0x8(%r11)
   1407963c7:	48 83 bc 24 60 03 00 	cmpq   $0x0,0x360(%rsp)
   1407963ce:	00 00 
   1407963d0:	74 2a                	je     0x1407963fc
   1407963d2:	48 83 bc 24 80 03 00 	cmpq   $0x0,0x380(%rsp)
   1407963d9:	00 00 
   1407963db:	75 1f                	jne    0x1407963fc
   1407963dd:	48 83 bc 24 20 03 00 	cmpq   $0x0,0x320(%rsp)
   1407963e4:	00 00 
   1407963e6:	74 0f                	je     0x1407963f7
   1407963e8:	48 83 bc 24 40 03 00 	cmpq   $0x0,0x340(%rsp)
   1407963ef:	00 00 
   1407963f1:	41 0f 94 c3          	sete   %r11b
   1407963f5:	eb 09                	jmp    0x140796400
   1407963f7:	45 31 db             	xor    %r11d,%r11d
   1407963fa:	eb 04                	jmp    0x140796400
   1407963fc:	45 31 db             	xor    %r11d,%r11d
   1407963ff:	90                   	nop
   140796400:	45 84 db             	test   %r11b,%r11b
   140796403:	74 33                	je     0x140796438
   140796405:	4c 8b 9c 24 50 03 00 	mov    0x350(%rsp),%r11
   14079640c:	00 
   14079640d:	4d 85 db             	test   %r11,%r11
   140796410:	7e 26                	jle    0x140796438
   140796412:	4c 8b a4 24 88 03 00 	mov    0x388(%rsp),%r12
   140796419:	00 
   14079641a:	4d 39 dc             	cmp    %r11,%r12
   14079641d:	75 19                	jne    0x140796438
   14079641f:	90                   	nop
   140796420:	4c 39 9c 24 10 03 00 	cmp    %r11,0x310(%rsp)
   140796427:	00 
   140796428:	75 0e                	jne    0x140796438
   14079642a:	4c 39 a4 24 48 03 00 	cmp    %r12,0x348(%rsp)
   140796431:	00 
   140796432:	41 0f 95 c3          	setne  %r11b
   140796436:	eb 1f                	jmp    0x140796457
   140796438:	45 31 db             	xor    %r11d,%r11d
   14079643b:	eb 1a                	jmp    0x140796457
   14079643d:	45 31 db             	xor    %r11d,%r11d
   140796440:	eb 15                	jmp    0x140796457
   140796442:	45 31 db             	xor    %r11d,%r11d
   140796445:	eb 10                	jmp    0x140796457
   140796447:	45 31 db             	xor    %r11d,%r11d
   14079644a:	eb 0b                	jmp    0x140796457
   14079644c:	4c 8b 94 24 e8 06 00 	mov    0x6e8(%rsp),%r10
   140796453:	00 
   140796454:	45 31 db             	xor    %r11d,%r11d
   140796457:	45 84 db             	test   %r11b,%r11b
   14079645a:	74 49                	je     0x1407964a5
   14079645c:	4c 8b 9c 24 50 06 00 	mov    0x650(%rsp),%r11
   140796463:	00 
   140796464:	4c 8b a4 24 40 01 00 	mov    0x140(%rsp),%r12
   14079646b:	00 
   14079646c:	4d 39 dc             	cmp    %r11,%r12
   14079646f:	0f 83 10 08 00 00    	jae    0x140796c85
   140796475:	4c 8b 9c 24 48 06 00 	mov    0x648(%rsp),%r11
   14079647c:	00 
   14079647d:	4c 8b ac 24 60 01 00 	mov    0x160(%rsp),%r13
   140796484:	00 
   140796485:	4b c7 84 2b 08 01 00 	movq   $0x0,0x108(%r11,%r13,1)
   14079648c:	00 00 00 00 00 
   140796491:	4c 8b 9c 24 68 06 00 	mov    0x668(%rsp),%r11
   140796498:	00 
   140796499:	45 31 ff             	xor    %r15d,%r15d
   14079649c:	0f 1f 40 00          	nopl   0x0(%rax)
   1407964a0:	e9 6d 07 00 00       	jmp    0x140796c12
   1407964a5:	48 83 fa 07          	cmp    $0x7,%rdx
   1407964a9:	e9 d0 00 00 00       	jmp    0x14079657e
   1407964ae:	90                   	nop
   1407964af:	90                   	nop
   1407964b0:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   1407964b5:	48 8b 94 24 d0 06 00 	mov    0x6d0(%rsp),%rdx
   1407964bc:	00 
   1407964bd:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   1407964c4:	00 
   1407964c5:	ff cb                	dec    %ebx
   1407964c7:	74 0d                	je     0x1407964d6
   1407964c9:	48 8b 84 24 78 06 00 	mov    0x678(%rsp),%rax
   1407964d0:	00 
   1407964d1:	e8 8a 25 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   1407964d6:	b8 10 00 00 00       	mov    $0x10,%eax
   1407964db:	48 8d 1d fe 68 bd 00 	lea    0xbd68fe(%rip),%rbx        # 0x14136cde0
   1407964e2:	b9 01 00 00 00       	mov    $0x1,%ecx
   1407964e7:	e8 14 c2 88 ff       	call   0x140022700 # runtime.mallocgcSmallScanNoHeaderSC2
   1407964ec:	48 c7 40 08 4e 00 00 	movq   $0x4e,0x8(%rax)
   1407964f3:	00 
   1407964f4:	48 8d 15 54 98 13 00 	lea    0x139854(%rip),%rdx        # 0x1408cfd4f
   1407964fb:	48 89 10             	mov    %rdx,(%rax)
   1407964fe:	48 8d 1d 9b b9 c7 00 	lea    0xc7b99b(%rip),%rbx        # 0x141411ea0
   140796505:	48 89 c1             	mov    %rax,%rcx
   140796508:	b8 99 01 00 00       	mov    $0x199,%eax
   14079650d:	48 81 c4 c0 06 00 00 	add    $0x6c0,%rsp
   140796514:	5d                   	pop    %rbp
   140796515:	c3                   	ret
   140796516:	90                   	nop
   140796517:	90                   	nop
   140796518:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   14079651d:	48 8b 94 24 d0 06 00 	mov    0x6d0(%rsp),%rdx
   140796524:	00 
   140796525:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   14079652c:	00 
   14079652d:	ff cb                	dec    %ebx
   14079652f:	74 0d                	je     0x14079653e
   140796531:	48 8b 84 24 78 06 00 	mov    0x678(%rsp),%rax
   140796538:	00 
   140796539:	e8 22 25 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   14079653e:	b8 10 00 00 00       	mov    $0x10,%eax
   140796543:	48 8d 1d 96 68 bd 00 	lea    0xbd6896(%rip),%rbx        # 0x14136cde0
   14079654a:	b9 01 00 00 00       	mov    $0x1,%ecx
   14079654f:	e8 ac c1 88 ff       	call   0x140022700 # runtime.mallocgcSmallScanNoHeaderSC2
   140796554:	48 c7 40 08 35 00 00 	movq   $0x35,0x8(%rax)
   14079655b:	00 
   14079655c:	48 8d 15 e5 c7 12 00 	lea    0x12c7e5(%rip),%rdx        # 0x1408c2d48
   140796563:	48 89 10             	mov    %rdx,(%rax)
   140796566:	48 8d 1d 33 b9 c7 00 	lea    0xc7b933(%rip),%rbx        # 0x141411ea0
   14079656d:	48 89 c1             	mov    %rax,%rcx
   140796570:	b8 94 01 00 00       	mov    $0x194,%eax
   140796575:	48 81 c4 c0 06 00 00 	add    $0x6c0,%rsp
   14079657c:	5d                   	pop    %rbp
   14079657d:	c3                   	ret
   14079657e:	48 89 9c 24 50 01 00 	mov    %rbx,0x150(%rsp)
   140796585:	00 
   140796586:	48 89 84 24 38 05 00 	mov    %rax,0x538(%rsp)
   14079658d:	00 
   14079658e:	48 89 84 24 f0 01 00 	mov    %rax,0x1f0(%rsp)
   140796595:	00 
   140796596:	48 89 9c 24 f8 01 00 	mov    %rbx,0x1f8(%rsp)
   14079659d:	00 
   14079659e:	66 90                	xchg   %ax,%ax
   1407965a0:	0f 85 e6 00 00 00    	jne    0x14079668c
   1407965a6:	41 81 3a 73 65 74 74 	cmpl   $0x74746573,(%r10)
   1407965ad:	0f 85 d9 00 00 00    	jne    0x14079668c
   1407965b3:	66 41 81 7a 04 69 6e 	cmpw   $0x6e69,0x4(%r10)
   1407965ba:	66 0f 1f 44 00 00    	nopw   0x0(%rax,%rax,1)
   1407965c0:	0f 85 c6 00 00 00    	jne    0x14079668c
   1407965c6:	41 80 7a 06 67       	cmpb   $0x67,0x6(%r10)
   1407965cb:	0f 85 bb 00 00 00    	jne    0x14079668c
   1407965d1:	48 8b 94 24 50 06 00 	mov    0x650(%rsp),%rdx
   1407965d8:	00 
   1407965d9:	4c 8b 94 24 40 01 00 	mov    0x140(%rsp),%r10
   1407965e0:	00 
   1407965e1:	49 39 d2             	cmp    %rdx,%r10
   1407965e4:	0f 83 20 06 00 00    	jae    0x140796c0a
   1407965ea:	48 8b 94 24 48 06 00 	mov    0x648(%rsp),%rdx
   1407965f1:	00 
   1407965f2:	4c 8b 9c 24 60 01 00 	mov    0x160(%rsp),%r11
   1407965f9:	00 
   1407965fa:	4a 8d 14 1a          	lea    (%rdx,%r11,1),%rdx
   1407965fe:	48 8d 92 48 01 00 00 	lea    0x148(%rdx),%rdx
   140796605:	83 3d 54 f3 2f 04 00 	cmpl   $0x0,0x42ff354(%rip)        # 0x144a95960
   14079660c:	74 47                	je     0x140796655
   14079660e:	48 89 94 24 80 06 00 	mov    %rdx,0x680(%rsp)
   140796615:	00 
   140796616:	48 8d 05 03 e1 c3 00 	lea    0xc3e103(%rip),%rax        # 0x1413d4720
   14079661d:	48 89 d3             	mov    %rdx,%rbx
   140796620:	48 8d 8c 24 c8 01 00 	lea    0x1c8(%rsp),%rcx
   140796627:	00 
   140796628:	e8 33 f7 88 ff       	call   0x140025d60 # runtime.wbMove
   14079662d:	48 8b 84 24 38 05 00 	mov    0x538(%rsp),%rax
   140796634:	00 
   140796635:	48 8b 94 24 80 06 00 	mov    0x680(%rsp),%rdx
   14079663c:	00 
   14079663d:	48 8b 9c 24 50 01 00 	mov    0x150(%rsp),%rbx
   140796644:	00 
   140796645:	4c 8b 94 24 40 01 00 	mov    0x140(%rsp),%r10
   14079664c:	00 
   14079664d:	4c 8b 9c 24 60 01 00 	mov    0x160(%rsp),%r11
   140796654:	00 
   140796655:	4c 8d a4 24 c8 01 00 	lea    0x1c8(%rsp),%r12
   14079665c:	00 
   14079665d:	45 0f 10 34 24       	movups (%r12),%xmm14
   140796662:	44 0f 11 32          	movups %xmm14,(%rdx)
   140796666:	45 0f 10 74 24 10    	movups 0x10(%r12),%xmm14
   14079666c:	44 0f 11 72 10       	movups %xmm14,0x10(%rdx)
   140796671:	45 0f 10 74 24 20    	movups 0x20(%r12),%xmm14
   140796677:	44 0f 11 72 20       	movups %xmm14,0x20(%rdx)
   14079667c:	45 0f 10 74 24 30    	movups 0x30(%r12),%xmm14
   140796682:	44 0f 11 72 30       	movups %xmm14,0x30(%rdx)
   140796687:	e9 ba 00 00 00       	jmp    0x140796746
   14079668c:	48 8b 94 24 50 06 00 	mov    0x650(%rsp),%rdx
   140796693:	00 
   140796694:	4c 8b 94 24 40 01 00 	mov    0x140(%rsp),%r10
   14079669b:	00 
   14079669c:	0f 1f 40 00          	nopl   0x0(%rax)
   1407966a0:	49 39 d2             	cmp    %rdx,%r10
   1407966a3:	0f 83 5c 05 00 00    	jae    0x140796c05
   1407966a9:	48 8b 94 24 48 06 00 	mov    0x648(%rsp),%rdx
   1407966b0:	00 
   1407966b1:	4c 8b 9c 24 60 01 00 	mov    0x160(%rsp),%r11
   1407966b8:	00 
   1407966b9:	4a 8d 14 1a          	lea    (%rdx,%r11,1),%rdx
   1407966bd:	48 8d 92 08 01 00 00 	lea    0x108(%rdx),%rdx
   1407966c4:	83 3d 95 f2 2f 04 00 	cmpl   $0x0,0x42ff295(%rip)        # 0x144a95960
   1407966cb:	74 47                	je     0x140796714
   1407966cd:	48 89 94 24 80 06 00 	mov    %rdx,0x680(%rsp)
   1407966d4:	00 
   1407966d5:	48 8d 05 44 e0 c3 00 	lea    0xc3e044(%rip),%rax        # 0x1413d4720
   1407966dc:	48 89 d3             	mov    %rdx,%rbx
   1407966df:	48 8d 8c 24 c8 01 00 	lea    0x1c8(%rsp),%rcx
   1407966e6:	00 
   1407966e7:	e8 74 f6 88 ff       	call   0x140025d60 # runtime.wbMove
   1407966ec:	48 8b 84 24 38 05 00 	mov    0x538(%rsp),%rax
   1407966f3:	00 
   1407966f4:	48 8b 94 24 80 06 00 	mov    0x680(%rsp),%rdx
   1407966fb:	00 
   1407966fc:	48 8b 9c 24 50 01 00 	mov    0x150(%rsp),%rbx
   140796703:	00 
   140796704:	4c 8b 94 24 40 01 00 	mov    0x140(%rsp),%r10
   14079670b:	00 
   14079670c:	4c 8b 9c 24 60 01 00 	mov    0x160(%rsp),%r11
   140796713:	00 
   140796714:	4c 8d a4 24 c8 01 00 	lea    0x1c8(%rsp),%r12
   14079671b:	00 
   14079671c:	45 0f 10 34 24       	movups (%r12),%xmm14
   140796721:	44 0f 11 32          	movups %xmm14,(%rdx)
   140796725:	45 0f 10 74 24 10    	movups 0x10(%r12),%xmm14
   14079672b:	44 0f 11 72 10       	movups %xmm14,0x10(%rdx)
   140796730:	45 0f 10 74 24 20    	movups 0x20(%r12),%xmm14
   140796736:	44 0f 11 72 20       	movups %xmm14,0x20(%rdx)
   14079673b:	45 0f 10 74 24 30    	movups 0x30(%r12),%xmm14
   140796741:	44 0f 11 72 30       	movups %xmm14,0x30(%rdx)
   140796746:	48 8b 94 24 50 06 00 	mov    0x650(%rsp),%rdx
   14079674d:	00 
   14079674e:	49 39 d2             	cmp    %rdx,%r10
   140796751:	0f 83 a7 04 00 00    	jae    0x140796bfe
   140796757:	48 8b 94 24 48 06 00 	mov    0x648(%rsp),%rdx
   14079675e:	00 
   14079675f:	4a ff 84 1a 90 01 00 	incq   0x190(%rdx,%r11,1)
   140796766:	00 
   140796767:	48 8b 94 24 50 06 00 	mov    0x650(%rsp),%rdx
   14079676e:	00 
   14079676f:	49 39 d2             	cmp    %rdx,%r10
   140796772:	0f 83 81 04 00 00    	jae    0x140796bf9
   140796778:	48 8b 94 24 48 06 00 	mov    0x648(%rsp),%rdx
   14079677f:	00 
   140796780:	4d 8d 24 13          	lea    (%r11,%rdx,1),%r12
   140796784:	4a 8b 94 1a 38 01 00 	mov    0x138(%rdx,%r11,1),%rdx
   14079678b:	00 
   14079678c:	4c 8d ac 24 a0 03 00 	lea    0x3a0(%rsp),%r13
   140796793:	00 
   140796794:	41 bf 06 00 00 00    	mov    $0x6,%r15d
   14079679a:	45 0f 10 34 24       	movups (%r12),%xmm14
   14079679f:	45 0f 11 75 00       	movups %xmm14,0x0(%r13)
   1407967a4:	45 0f 10 74 24 10    	movups 0x10(%r12),%xmm14
   1407967aa:	45 0f 11 75 10       	movups %xmm14,0x10(%r13)
   1407967af:	45 0f 10 74 24 20    	movups 0x20(%r12),%xmm14
   1407967b5:	45 0f 11 75 20       	movups %xmm14,0x20(%r13)
   1407967ba:	45 0f 10 74 24 30    	movups 0x30(%r12),%xmm14
   1407967c0:	45 0f 11 75 30       	movups %xmm14,0x30(%r13)
   1407967c5:	49 83 c4 40          	add    $0x40,%r12
   1407967c9:	49 83 c5 40          	add    $0x40,%r13
   1407967cd:	41 ff cf             	dec    %r15d
   1407967d0:	75 c8                	jne    0x14079679a
   1407967d2:	45 0f 10 34 24       	movups (%r12),%xmm14
   1407967d7:	45 0f 11 75 00       	movups %xmm14,0x0(%r13)
   1407967dc:	45 0f 10 74 24 08    	movups 0x8(%r12),%xmm14
   1407967e2:	45 0f 11 75 08       	movups %xmm14,0x8(%r13)
   1407967e7:	48 85 d2             	test   %rdx,%rdx
   1407967ea:	74 11                	je     0x1407967fd
   1407967ec:	31 c9                	xor    %ecx,%ecx
   1407967ee:	48 8d 94 24 68 01 00 	lea    0x168(%rsp),%rdx
   1407967f5:	00 
   1407967f6:	31 f6                	xor    %esi,%esi
   1407967f8:	e9 82 00 00 00       	jmp    0x14079687f
   1407967fd:	48 8d 84 24 68 01 00 	lea    0x168(%rsp),%rax
   140796804:	00 
   140796805:	bb 01 00 00 00       	mov    $0x1,%ebx
   14079680a:	31 c9                	xor    %ecx,%ecx
   14079680c:	89 df                	mov    %ebx,%edi
   14079680e:	48 8d 35 ab 6e bb 00 	lea    0xbb6eab(%rip),%rsi        # 0x14134d6c0
   140796815:	4c 8d 84 24 a0 06 00 	lea    0x6a0(%rsp),%r8
   14079681c:	00 
   14079681d:	41 b9 02 00 00 00    	mov    $0x2,%r9d
   140796823:	e8 78 07 8d ff       	call   0x140066fa0 # runtime.growsliceBuf
   140796828:	48 8d 53 ff          	lea    -0x1(%rbx),%rdx
   14079682c:	48 c1 e2 04          	shl    $0x4,%rdx
   140796830:	48 c7 44 10 08 08 00 	movq   $0x8,0x8(%rax,%rdx,1)
   140796837:	00 00 
   140796839:	83 3d 20 f1 2f 04 00 	cmpl   $0x0,0x42ff120(%rip)        # 0x144a95960
   140796840:	74 0c                	je     0x14079684e
   140796842:	4c 8b 14 10          	mov    (%rax,%rdx,1),%r10
   140796846:	e8 75 75 8f ff       	call   0x14008ddc0 # runtime.gcWriteBarrier1
   14079684b:	4d 89 13             	mov    %r10,(%r11)
   14079684e:	4c 8d 15 0b 25 0f 00 	lea    0xf250b(%rip),%r10        # 0x140888d60
   140796855:	4c 89 14 10          	mov    %r10,(%rax,%rdx,1)
   140796859:	4c 8b 94 24 40 01 00 	mov    0x140(%rsp),%r10
   140796860:	00 
   140796861:	4c 8b 9c 24 60 01 00 	mov    0x160(%rsp),%r11
   140796868:	00 
   140796869:	48 89 c2             	mov    %rax,%rdx
   14079686c:	48 89 de             	mov    %rbx,%rsi
   14079686f:	48 8b 84 24 38 05 00 	mov    0x538(%rsp),%rax
   140796876:	00 
   140796877:	48 8b 9c 24 50 01 00 	mov    0x150(%rsp),%rbx
   14079687e:	00 
   14079687f:	48 83 bc 24 18 05 00 	cmpq   $0x0,0x518(%rsp)
   140796886:	00 00 
   140796888:	0f 85 8e 00 00 00    	jne    0x14079691c
   14079688e:	4c 8d 66 01          	lea    0x1(%rsi),%r12
   140796892:	4c 39 e1             	cmp    %r12,%rcx
   140796895:	73 4b                	jae    0x1407968e2
   140796897:	48 89 d0             	mov    %rdx,%rax
   14079689a:	4c 89 e3             	mov    %r12,%rbx
   14079689d:	bf 01 00 00 00       	mov    $0x1,%edi
   1407968a2:	48 8d 35 17 6e bb 00 	lea    0xbb6e17(%rip),%rsi        # 0x14134d6c0
   1407968a9:	4c 8d 84 24 a0 06 00 	lea    0x6a0(%rsp),%r8
   1407968b0:	00 
   1407968b1:	41 b9 02 00 00 00    	mov    $0x2,%r9d
   1407968b7:	e8 e4 06 8d ff       	call   0x140066fa0 # runtime.growsliceBuf
   1407968bc:	4c 8b 94 24 40 01 00 	mov    0x140(%rsp),%r10
   1407968c3:	00 
   1407968c4:	4c 8b 9c 24 60 01 00 	mov    0x160(%rsp),%r11
   1407968cb:	00 
   1407968cc:	48 89 c2             	mov    %rax,%rdx
   1407968cf:	49 89 dc             	mov    %rbx,%r12
   1407968d2:	48 8b 84 24 38 05 00 	mov    0x538(%rsp),%rax
   1407968d9:	00 
   1407968da:	48 8b 9c 24 50 01 00 	mov    0x150(%rsp),%rbx
   1407968e1:	00 
   1407968e2:	49 8d 74 24 ff       	lea    -0x1(%r12),%rsi
   1407968e7:	48 c1 e6 04          	shl    $0x4,%rsi
   1407968eb:	48 c7 44 32 08 07 00 	movq   $0x7,0x8(%rdx,%rsi,1)
   1407968f2:	00 00 
   1407968f4:	83 3d 65 f0 2f 04 00 	cmpl   $0x0,0x42ff065(%rip)        # 0x144a95960
   1407968fb:	74 12                	je     0x14079690f
   1407968fd:	4c 8b 04 32          	mov    (%rdx,%rsi,1),%r8
   140796901:	4c 89 df             	mov    %r11,%rdi
   140796904:	e8 b7 74 8f ff       	call   0x14008ddc0 # runtime.gcWriteBarrier1
   140796909:	4d 89 03             	mov    %r8,(%r11)
   14079690c:	49 89 fb             	mov    %rdi,%r11
   14079690f:	4c 8d 05 84 05 0f 00 	lea    0xf0584(%rip),%r8        # 0x140886e9a
   140796916:	4c 89 04 32          	mov    %r8,(%rdx,%rsi,1)
   14079691a:	eb 03                	jmp    0x14079691f
   14079691c:	49 89 f4             	mov    %rsi,%r12
   14079691f:	48 8d b4 24 d0 06 00 	lea    0x6d0(%rsp),%rsi
   140796926:	00 
   140796927:	48 29 e6             	sub    %rsp,%rsi
   14079692a:	49 89 d0             	mov    %rdx,%r8
   14079692d:	48 29 e2             	sub    %rsp,%rdx
   140796930:	48 39 f2             	cmp    %rsi,%rdx
   140796933:	73 38                	jae    0x14079696d
   140796935:	48 8d 05 84 6d bb 00 	lea    0xbb6d84(%rip),%rax        # 0x14134d6c0
   14079693c:	4c 89 c3             	mov    %r8,%rbx
   14079693f:	48 89 cf             	mov    %rcx,%rdi
   140796942:	4c 89 e1             	mov    %r12,%rcx
   140796945:	e8 16 01 8d ff       	call   0x140066a60 # runtime.moveSlice
   14079694a:	48 8b 84 24 38 05 00 	mov    0x538(%rsp),%rax
   140796951:	00 
   140796952:	4c 8b 94 24 40 01 00 	mov    0x140(%rsp),%r10
   140796959:	00 
   14079695a:	4c 8b 9c 24 60 01 00 	mov    0x160(%rsp),%r11
   140796961:	00 
   140796962:	49 89 dc             	mov    %rbx,%r12
   140796965:	48 8b 9c 24 50 01 00 	mov    0x150(%rsp),%rbx
   14079696c:	00 
   14079696d:	4d 85 e4             	test   %r12,%r12
   140796970:	75 74                	jne    0x1407969e6
   140796972:	48 8b 8c 24 50 06 00 	mov    0x650(%rsp),%rcx
   140796979:	00 
   14079697a:	66 0f 1f 44 00 00    	nopw   0x0(%rax,%rax,1)
   140796980:	49 39 ca             	cmp    %rcx,%r10
   140796983:	0f 83 6b 02 00 00    	jae    0x140796bf4
   140796989:	48 8b 8c 24 48 06 00 	mov    0x648(%rsp),%rcx
   140796990:	00 
   140796991:	42 c6 84 19 e0 00 00 	movb   $0x0,0xe0(%rcx,%r11,1)
   140796998:	00 00 
   14079699a:	48 8b 8c 24 50 06 00 	mov    0x650(%rsp),%rcx
   1407969a1:	00 
   1407969a2:	49 39 ca             	cmp    %rcx,%r10
   1407969a5:	0f 83 44 02 00 00    	jae    0x140796bef
   1407969ab:	48 8b 8c 24 48 06 00 	mov    0x648(%rsp),%rcx
   1407969b2:	00 
   1407969b3:	4a 89 9c 19 00 01 00 	mov    %rbx,0x100(%rcx,%r11,1)
   1407969ba:	00 
   1407969bb:	83 3d 9e ef 2f 04 00 	cmpl   $0x0,0x42fef9e(%rip)        # 0x144a95960
   1407969c2:	74 1a                	je     0x1407969de
   1407969c4:	4a 8b 94 19 f8 00 00 	mov    0xf8(%rcx,%r11,1),%rdx
   1407969cb:	00 
   1407969cc:	4c 89 de             	mov    %r11,%rsi
   1407969cf:	e8 0c 74 8f ff       	call   0x14008dde0 # runtime.gcWriteBarrier2
   1407969d4:	49 89 03             	mov    %rax,(%r11)
   1407969d7:	49 89 53 08          	mov    %rdx,0x8(%r11)
   1407969db:	49 89 f3             	mov    %rsi,%r11
   1407969de:	4a 89 84 19 f8 00 00 	mov    %rax,0xf8(%rcx,%r11,1)
   1407969e5:	00 
   1407969e6:	48 8b 8c 24 68 06 00 	mov    0x668(%rsp),%rcx
   1407969ed:	00 
   1407969ee:	48 8b 94 24 48 01 00 	mov    0x148(%rsp),%rdx
   1407969f5:	00 
   1407969f6:	48 39 ca             	cmp    %rcx,%rdx
   1407969f9:	0f 83 eb 01 00 00    	jae    0x140796bea
   1407969ff:	48 8b 8c 24 60 06 00 	mov    0x660(%rsp),%rcx
   140796a06:	00 
   140796a07:	48 8b b4 24 58 01 00 	mov    0x158(%rsp),%rsi
   140796a0e:	00 
   140796a0f:	c6 44 31 08 00       	movb   $0x0,0x8(%rcx,%rsi,1)
   140796a14:	48 8b 8c 24 68 06 00 	mov    0x668(%rsp),%rcx
   140796a1b:	00 
   140796a1c:	0f 1f 40 00          	nopl   0x0(%rax)
   140796a20:	48 39 ca             	cmp    %rcx,%rdx
   140796a23:	0f 83 bc 01 00 00    	jae    0x140796be5
   140796a29:	48 8b 8c 24 60 06 00 	mov    0x660(%rsp),%rcx
   140796a30:	00 
   140796a31:	48 89 9c 31 a0 00 00 	mov    %rbx,0xa0(%rcx,%rsi,1)
   140796a38:	00 
   140796a39:	83 3d 20 ef 2f 04 00 	cmpl   $0x0,0x42fef20(%rip)        # 0x144a95960
   140796a40:	74 14                	je     0x140796a56
   140796a42:	48 8b 9c 31 98 00 00 	mov    0x98(%rcx,%rsi,1),%rbx
   140796a49:	00 
   140796a4a:	e8 91 73 8f ff       	call   0x14008dde0 # runtime.gcWriteBarrier2
   140796a4f:	49 89 03             	mov    %rax,(%r11)
   140796a52:	49 89 5b 08          	mov    %rbx,0x8(%r11)
   140796a56:	48 89 84 31 98 00 00 	mov    %rax,0x98(%rcx,%rsi,1)
   140796a5d:	00 
   140796a5e:	48 8b 8c 24 68 06 00 	mov    0x668(%rsp),%rcx
   140796a65:	00 
   140796a66:	48 39 ca             	cmp    %rcx,%rdx
   140796a69:	0f 83 6d 01 00 00    	jae    0x140796bdc
   140796a6f:	48 8b 8c 24 60 06 00 	mov    0x660(%rsp),%rcx
   140796a76:	00 
   140796a77:	48 ff 44 31 40       	incq   0x40(%rcx,%rsi,1)
   140796a7c:	48 89 e1             	mov    %rsp,%rcx
   140796a7f:	48 8d 94 24 40 05 00 	lea    0x540(%rsp),%rdx
   140796a86:	00 
   140796a87:	bb 04 00 00 00       	mov    $0x4,%ebx
   140796a8c:	44 0f 10 32          	movups (%rdx),%xmm14
   140796a90:	44 0f 11 31          	movups %xmm14,(%rcx)
   140796a94:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   140796a99:	44 0f 11 71 10       	movups %xmm14,0x10(%rcx)
   140796a9e:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   140796aa3:	44 0f 11 71 20       	movups %xmm14,0x20(%rcx)
   140796aa8:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   140796aad:	44 0f 11 71 30       	movups %xmm14,0x30(%rcx)
   140796ab2:	48 83 c2 40          	add    $0x40,%rdx
   140796ab6:	48 83 c1 40          	add    $0x40,%rcx
   140796aba:	ff cb                	dec    %ebx
   140796abc:	75 ce                	jne    0x140796a8c
   140796abe:	44 0f 10 32          	movups (%rdx),%xmm14
   140796ac2:	44 0f 11 31          	movups %xmm14,(%rcx)
   140796ac6:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   140796acb:	44 0f 11 71 10       	movups %xmm14,0x10(%rcx)
   140796ad0:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   140796ad5:	44 0f 11 71 20       	movups %xmm14,0x20(%rcx)
   140796ada:	44 0f 10 72 28       	movups 0x28(%rdx),%xmm14
   140796adf:	44 0f 11 71 28       	movups %xmm14,0x28(%rcx)
   140796ae4:	48 8b 84 24 d0 06 00 	mov    0x6d0(%rsp),%rax
   140796aeb:	00 
   140796aec:	e8 6f 59 ff ff       	call   0x14078c460 # main.(*Host).commit
   140796af1:	48 85 c0             	test   %rax,%rax
   140796af4:	74 68                	je     0x140796b5e
   140796af6:	90                   	nop
   140796af7:	90                   	nop
   140796af8:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   140796afd:	48 8b 94 24 d0 06 00 	mov    0x6d0(%rsp),%rdx
   140796b04:	00 
   140796b05:	f0 0f c1 9a d0 00 00 	lock xadd %ebx,0xd0(%rdx)
   140796b0c:	00 
   140796b0d:	ff cb                	dec    %ebx
   140796b0f:	74 0d                	je     0x140796b1e
   140796b11:	48 8b 84 24 78 06 00 	mov    0x678(%rsp),%rax
   140796b18:	00 
   140796b19:	e8 42 1f 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   140796b1e:	b8 10 00 00 00       	mov    $0x10,%eax
   140796b23:	48 8d 1d b6 62 bd 00 	lea    0xbd62b6(%rip),%rbx        # 0x14136cde0
   140796b2a:	b9 01 00 00 00       	mov    $0x1,%ecx
   140796b2f:	e8 cc bb 88 ff       	call   0x140022700 # runtime.mallocgcSmallScanNoHeaderSC2
   140796b34:	48 c7 40 08 4f 00 00 	movq   $0x4f,0x8(%rax)
   140796b3b:	00 
   140796b3c:	48 8d 15 8c 97 13 00 	lea    0x13978c(%rip),%rdx        # 0x1408d02cf
   140796b43:	48 89 10             	mov    %rdx,(%rax)
   140796b46:	48 8d 1d 53 b3 c7 00 	lea    0xc7b353(%rip),%rbx        # 0x141411ea0
   140796b4d:	48 89 c1             	mov    %rax,%rcx
   140796b50:	b8 f7 01 00 00       	mov    $0x1f7,%eax
   140796b55:	48 81 c4 c0 06 00 00 	add    $0x6c0,%rsp
   140796b5c:	5d                   	pop    %rbp
   140796b5d:	c3                   	ret
   140796b5e:	90                   	nop
   140796b5f:	90                   	nop
   140796b60:	bb ff ff ff ff       	mov    $0xffffffff,%ebx
   140796b65:	48 8b b4 24 d0 06 00 	mov    0x6d0(%rsp),%rsi
   140796b6c:	00 
   140796b6d:	f0 0f c1 9e d0 00 00 	lock xadd %ebx,0xd0(%rsi)
   140796b74:	00 
   140796b75:	ff cb                	dec    %ebx
   140796b77:	74 15                	je     0x140796b8e
   140796b79:	48 8b 84 24 78 06 00 	mov    0x678(%rsp),%rax
   140796b80:	00 
   140796b81:	e8 da 1e 90 ff       	call   0x140098a60 # internal/sync.(*Mutex).unlockSlow
   140796b86:	48 8b b4 24 d0 06 00 	mov    0x6d0(%rsp),%rsi
   140796b8d:	00 
   140796b8e:	48 8b be c0 00 00 00 	mov    0xc0(%rsi),%rdi
   140796b95:	48 85 ff             	test   %rdi,%rdi
   140796b98:	74 1a                	je     0x140796bb4
   140796b9a:	48 8b 47 78          	mov    0x78(%rdi),%rax
   140796b9e:	90                   	nop
   140796b9f:	48 8d 9c 24 40 01 00 	lea    0x140(%rsp),%rbx
   140796ba6:	00 
   140796ba7:	e8 94 3b 88 ff       	call   0x14001a740 # runtime.selectnbsend
   140796bac:	48 8b b4 24 d0 06 00 	mov    0x6d0(%rsp),%rsi
   140796bb3:	00 
   140796bb4:	48 8b 96 c0 02 00 00 	mov    0x2c0(%rsi),%rdx
   140796bbb:	0f 1f 44 00 00       	nopl   0x0(%rax,%rax,1)
   140796bc0:	48 85 d2             	test   %rdx,%rdx
   140796bc3:	74 05                	je     0x140796bca
   140796bc5:	48 8b 02             	mov    (%rdx),%rax
   140796bc8:	ff d0                	call   *%rax
   140796bca:	b8 c8 00 00 00       	mov    $0xc8,%eax
   140796bcf:	31 db                	xor    %ebx,%ebx
   140796bd1:	31 c9                	xor    %ecx,%ecx
   140796bd3:	48 81 c4 c0 06 00 00 	add    $0x6c0,%rsp
   140796bda:	5d                   	pop    %rbp
   140796bdb:	c3                   	ret
   140796bdc:	0f 1f 40 00          	nopl   0x0(%rax)
   140796be0:	e8 9b 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796be5:	e8 96 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796bea:	e8 91 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796bef:	e8 8c 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796bf4:	e8 87 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796bf9:	e8 82 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796bfe:	66 90                	xchg   %ax,%ax
   140796c00:	e8 7b 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796c05:	e8 76 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796c0a:	e8 71 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796c0f:	49 ff c7             	inc    %r15
   140796c12:	4d 39 df             	cmp    %r11,%r15
   140796c15:	7d 57                	jge    0x140796c6e
   140796c17:	48 8b 8c 24 68 06 00 	mov    0x668(%rsp),%rcx
   140796c1e:	00 
   140796c1f:	90                   	nop
   140796c20:	49 39 cf             	cmp    %rcx,%r15
   140796c23:	73 57                	jae    0x140796c7c
   140796c25:	48 8b 8c 24 60 06 00 	mov    0x660(%rsp),%rcx
   140796c2c:	00 
   140796c2d:	49 69 f7 b8 00 00 00 	imul   $0xb8,%r15,%rsi
   140796c34:	48 8b 7c 31 60       	mov    0x60(%rcx,%rsi,1),%rdi
   140796c39:	0f 1f 80 00 00 00 00 	nopl   0x0(%rax)
   140796c40:	48 39 bc 24 e0 04 00 	cmp    %rdi,0x4e0(%rsp)
   140796c47:	00 
   140796c48:	75 c5                	jne    0x140796c0f
   140796c4a:	48 c7 04 31 00 00 00 	movq   $0x0,(%rcx,%rsi,1)
   140796c51:	00 
   140796c52:	48 8b 8c 24 68 06 00 	mov    0x668(%rsp),%rcx
   140796c59:	00 
   140796c5a:	49 39 cf             	cmp    %rcx,%r15
   140796c5d:	73 18                	jae    0x140796c77
   140796c5f:	48 8b 8c 24 60 06 00 	mov    0x660(%rsp),%rcx
   140796c66:	00 
   140796c67:	48 ff 44 31 40       	incq   0x40(%rcx,%rsi,1)
   140796c6c:	eb a1                	jmp    0x140796c0f
   140796c6e:	48 83 fa 07          	cmp    $0x7,%rdx
   140796c72:	e9 07 f9 ff ff       	jmp    0x14079657e
   140796c77:	e8 04 75 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796c7c:	0f 1f 40 00          	nopl   0x0(%rax)
   140796c80:	e8 fb 74 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796c85:	e8 f6 74 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796c8a:	e8 f1 74 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796c8f:	e8 ec 74 8f ff       	call   0x14008e180 # runtime.panicBounds
   140796c94:	90                   	nop
   140796c95:	48 89 44 24 08       	mov    %rax,0x8(%rsp)
   140796c9a:	48 89 5c 24 10       	mov    %rbx,0x10(%rsp)
   140796c9f:	48 89 4c 24 18       	mov    %rcx,0x18(%rsp)
   140796ca4:	48 89 7c 24 20       	mov    %rdi,0x20(%rsp)
   140796ca9:	48 89 74 24 28       	mov    %rsi,0x28(%rsp)
   140796cae:	e8 8d 56 8f ff       	call   0x14008c340 # runtime.morestack_noctxt
   140796cb3:	48 8b 44 24 08       	mov    0x8(%rsp),%rax
   140796cb8:	48 8b 5c 24 10       	mov    0x10(%rsp),%rbx
   140796cbd:	48 8b 4c 24 18       	mov    0x18(%rsp),%rcx
   140796cc2:	48 8b 7c 24 20       	mov    0x20(%rsp),%rdi
   140796cc7:	48 8b 74 24 28       	mov    0x28(%rsp),%rsi
   140796ccc:	e9 8f ed ff ff       	jmp    0x140795a60
   140796cd1:	cc                   	int3
   140796cd2:	cc                   	int3
   140796cd3:	cc                   	int3
   140796cd4:	cc                   	int3
   140796cd5:	cc                   	int3
   140796cd6:	cc                   	int3
   140796cd7:	cc                   	int3
   140796cd8:	cc                   	int3
   140796cd9:	cc                   	int3
   140796cda:	cc                   	int3
   140796cdb:	cc                   	int3
   140796cdc:	cc                   	int3
   140796cdd:	cc                   	int3
   140796cde:	cc                   	int3
   140796cdf:	cc                   	int3

# main.(*Host).addCombinedRoutes.func1 @ 0x1407e5460
   1407e5460:	4c 8d a4 24 08 fa ff 	lea    -0x5f8(%rsp),%r12
   1407e5467:	ff 
   1407e5468:	4d 3b 66 10          	cmp    0x10(%r14),%r12
   1407e546c:	0f 86 4c 11 00 00    	jbe    0x1407e65be
   1407e5472:	55                   	push   %rbp
   1407e5473:	48 89 e5             	mov    %rsp,%rbp
   1407e5476:	48 81 ec 70 06 00 00 	sub    $0x670,%rsp
   1407e547d:	48 89 8c 24 90 06 00 	mov    %rcx,0x690(%rsp)
   1407e5484:	00 
   1407e5485:	48 89 84 24 80 06 00 	mov    %rax,0x680(%rsp)
   1407e548c:	00 
   1407e548d:	48 89 9c 24 88 06 00 	mov    %rbx,0x688(%rsp)
   1407e5494:	00 
   1407e5495:	48 8b 52 08          	mov    0x8(%rdx),%rdx
   1407e5499:	48 89 94 24 28 04 00 	mov    %rdx,0x428(%rsp)
   1407e54a0:	00 
   1407e54a1:	e8 7a 83 fa ff       	call   0x14078d820 # main.parsePost
   1407e54a6:	84 c0                	test   %al,%al
   1407e54a8:	0f 84 00 02 00 00    	je     0x1407e56ae
   1407e54ae:	48 8b 84 24 28 04 00 	mov    0x428(%rsp),%rax
   1407e54b5:	00 
   1407e54b6:	84 00                	test   %al,(%rax)
   1407e54b8:	48 05 d0 00 00 00    	add    $0xd0,%rax
   1407e54be:	48 89 84 24 78 04 00 	mov    %rax,0x478(%rsp)
   1407e54c5:	00 
   1407e54c6:	e8 75 3d 8b ff       	call   0x140099240 # sync.(*Mutex).Lock
   1407e54cb:	48 8b 84 24 90 06 00 	mov    0x690(%rsp),%rax
   1407e54d2:	00 
   1407e54d3:	48 8d 1d a9 d3 09 00 	lea    0x9d3a9(%rip),%rbx        # 0x140882883
   1407e54da:	b9 05 00 00 00       	mov    $0x5,%ecx
   1407e54df:	90                   	nop
   1407e54e0:	e8 3b ca b9 ff       	call   0x140381f20 # net/http.(*Request).PathValue
   1407e54e5:	48 8b 8c 24 28 04 00 	mov    0x428(%rsp),%rcx
   1407e54ec:	00 
   1407e54ed:	48 8b 91 b0 01 00 00 	mov    0x1b0(%rcx),%rdx
   1407e54f4:	48 8b b9 b8 01 00 00 	mov    0x1b8(%rcx),%rdi
   1407e54fb:	48 89 d1             	mov    %rdx,%rcx
   1407e54fe:	66 90                	xchg   %ax,%ax
   1407e5500:	e8 5b 81 fa ff       	call   0x14078d660 # main.equal
   1407e5505:	84 c0                	test   %al,%al
   1407e5507:	0f 84 67 01 00 00    	je     0x1407e5674
   1407e550d:	48 8b 94 24 90 06 00 	mov    0x690(%rsp),%rdx
   1407e5514:	00 
   1407e5515:	48 8b 82 98 00 00 00 	mov    0x98(%rdx),%rax
   1407e551c:	48 8d 1d 70 19 0a 00 	lea    0xa1970(%rip),%rbx        # 0x140886e93
   1407e5523:	b9 07 00 00 00       	mov    $0x7,%ecx
   1407e5528:	e8 b3 b9 a6 ff       	call   0x140250ee0 # net/url.Values.Get
   1407e552d:	b9 0a 00 00 00       	mov    $0xa,%ecx
   1407e5532:	bf 40 00 00 00       	mov    $0x40,%edi
   1407e5537:	e8 44 64 8e ff       	call   0x1400cb980 # strconv.ParseInt
   1407e553c:	48 89 84 24 58 01 00 	mov    %rax,0x158(%rsp)
   1407e5543:	00 
   1407e5544:	48 89 9c 24 70 01 00 	mov    %rbx,0x170(%rsp)
   1407e554b:	00 
   1407e554c:	48 8b 94 24 90 06 00 	mov    0x690(%rsp),%rdx
   1407e5553:	00 
   1407e5554:	48 8b 82 98 00 00 00 	mov    0x98(%rdx),%rax
   1407e555b:	48 8d 1d 51 f3 09 00 	lea    0x9f351(%rip),%rbx        # 0x1408848b3
   1407e5562:	b9 06 00 00 00       	mov    $0x6,%ecx
   1407e5567:	e8 74 b9 a6 ff       	call   0x140250ee0 # net/url.Values.Get
   1407e556c:	e8 cf d6 8f ff       	call   0x1400e2c40 # strings.TrimSpace
   1407e5571:	48 89 84 24 20 04 00 	mov    %rax,0x420(%rsp)
   1407e5578:	00 
   1407e5579:	48 89 9c 24 50 01 00 	mov    %rbx,0x150(%rsp)
   1407e5580:	00 
   1407e5581:	48 8b 94 24 90 06 00 	mov    0x690(%rsp),%rdx
   1407e5588:	00 
   1407e5589:	48 8b 82 98 00 00 00 	mov    0x98(%rdx),%rax
   1407e5590:	48 8d 1d 6a 56 0b 00 	lea    0xb566a(%rip),%rbx        # 0x14089ac01
   1407e5597:	b9 13 00 00 00       	mov    $0x13,%ecx
   1407e559c:	0f 1f 40 00          	nopl   0x0(%rax)
   1407e55a0:	e8 3b b9 a6 ff       	call   0x140250ee0 # net/url.Values.Get
   1407e55a5:	48 89 84 24 30 04 00 	mov    %rax,0x430(%rsp)
   1407e55ac:	00 
   1407e55ad:	48 89 9c 24 68 01 00 	mov    %rbx,0x168(%rsp)
   1407e55b4:	00 
   1407e55b5:	48 89 d9             	mov    %rbx,%rcx
   1407e55b8:	48 8b bc 24 58 01 00 	mov    0x158(%rsp),%rdi
   1407e55bf:	00 
   1407e55c0:	48 8d 35 91 37 0a 00 	lea    0xa3791(%rip),%rsi        # 0x140888d58
   1407e55c7:	41 b8 08 00 00 00    	mov    $0x8,%r8d
   1407e55cd:	48 89 c3             	mov    %rax,%rbx
   1407e55d0:	48 8b 84 24 28 04 00 	mov    0x428(%rsp),%rax
   1407e55d7:	00 
   1407e55d8:	e8 83 87 fa ff       	call   0x14078dd60 # main.(*Host).verifyForm
   1407e55dd:	48 8b 94 24 70 01 00 	mov    0x170(%rsp),%rdx
   1407e55e4:	00 
   1407e55e5:	48 85 d2             	test   %rdx,%rdx
   1407e55e8:	75 4d                	jne    0x1407e5637
   1407e55ea:	84 db                	test   %bl,%bl
   1407e55ec:	74 49                	je     0x1407e5637
   1407e55ee:	48 89 84 24 60 01 00 	mov    %rax,0x160(%rsp)
   1407e55f5:	00 
   1407e55f6:	48 8b 84 24 20 04 00 	mov    0x420(%rsp),%rax
   1407e55fd:	00 
   1407e55fe:	48 8b 9c 24 50 01 00 	mov    0x150(%rsp),%rbx
   1407e5605:	00 
   1407e5606:	b9 28 00 00 00       	mov    $0x28,%ecx
   1407e560b:	e8 d0 80 fa ff       	call   0x14078d6e0 # main.cleanField
   1407e5610:	84 c0                	test   %al,%al
   1407e5612:	74 23                	je     0x1407e5637
   1407e5614:	48 8b 84 24 28 04 00 	mov    0x428(%rsp),%rax
   1407e561b:	00 
   1407e561c:	48 8b 90 f8 01 00 00 	mov    0x1f8(%rax),%rdx
   1407e5623:	48 8b b0 00 02 00 00 	mov    0x200(%rax),%rsi
   1407e562a:	48 8b bc 24 68 01 00 	mov    0x168(%rsp),%rdi
   1407e5631:	00 
   1407e5632:	e9 8a 00 00 00       	jmp    0x1407e56c1
   1407e5637:	48 8b 84 24 78 04 00 	mov    0x478(%rsp),%rax
   1407e563e:	00 
   1407e563f:	90                   	nop
   1407e5640:	e8 9b 3c 8b ff       	call   0x1400992e0 # sync.(*Mutex).Unlock
   1407e5645:	48 8b 84 24 80 06 00 	mov    0x680(%rsp),%rax
   1407e564c:	00 
   1407e564d:	48 8b 9c 24 88 06 00 	mov    0x688(%rsp),%rbx
   1407e5654:	00 
   1407e5655:	48 8d 0d 1f 5e 0e 00 	lea    0xe5e1f(%rip),%rcx        # 0x1408cb47b
   1407e565c:	bf 42 00 00 00       	mov    $0x42,%edi
   1407e5661:	be 90 01 00 00       	mov    $0x190,%esi
   1407e5666:	e8 75 a3 ba ff       	call   0x14038f9e0 # net/http.Error
   1407e566b:	48 81 c4 70 06 00 00 	add    $0x670,%rsp
   1407e5672:	5d                   	pop    %rbp
   1407e5673:	c3                   	ret
   1407e5674:	48 8b 84 24 78 04 00 	mov    0x478(%rsp),%rax
   1407e567b:	00 
   1407e567c:	0f 1f 40 00          	nopl   0x0(%rax)
   1407e5680:	e8 5b 3c 8b ff       	call   0x1400992e0 # sync.(*Mutex).Unlock
   1407e5685:	48 8b 84 24 80 06 00 	mov    0x680(%rsp),%rax
   1407e568c:	00 
   1407e568d:	48 8b 9c 24 88 06 00 	mov    0x688(%rsp),%rbx
   1407e5694:	00 
   1407e5695:	48 8b 8c 24 90 06 00 	mov    0x690(%rsp),%rcx
   1407e569c:	00 
   1407e569d:	0f 1f 00             	nopl   (%rax)
   1407e56a0:	e8 fb a5 ba ff       	call   0x14038fca0 # net/http.NotFound
   1407e56a5:	48 81 c4 70 06 00 00 	add    $0x670,%rsp
   1407e56ac:	5d                   	pop    %rbp
   1407e56ad:	c3                   	ret
   1407e56ae:	48 81 c4 70 06 00 00 	add    $0x670,%rsp
   1407e56b5:	5d                   	pop    %rbp
   1407e56b6:	c3                   	ret
   1407e56b7:	48 81 c2 b8 00 00 00 	add    $0xb8,%rdx
   1407e56be:	48 ff ce             	dec    %rsi
   1407e56c1:	48 85 f6             	test   %rsi,%rsi
   1407e56c4:	0f 8e ad 01 00 00    	jle    0x1407e5877
   1407e56ca:	48 8b 8a b0 00 00 00 	mov    0xb0(%rdx),%rcx
   1407e56d1:	4c 8b 82 a8 00 00 00 	mov    0xa8(%rdx),%r8
   1407e56d8:	4c 8d 8c 24 e8 02 00 	lea    0x2e8(%rsp),%r9
   1407e56df:	00 
   1407e56e0:	44 0f 10 32          	movups (%rdx),%xmm14
   1407e56e4:	45 0f 11 31          	movups %xmm14,(%r9)
   1407e56e8:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   1407e56ed:	45 0f 11 71 10       	movups %xmm14,0x10(%r9)
   1407e56f2:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   1407e56f7:	45 0f 11 71 20       	movups %xmm14,0x20(%r9)
   1407e56fc:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   1407e5701:	45 0f 11 71 30       	movups %xmm14,0x30(%r9)
   1407e5706:	44 0f 10 72 40       	movups 0x40(%rdx),%xmm14
   1407e570b:	45 0f 11 71 40       	movups %xmm14,0x40(%r9)
   1407e5710:	44 0f 10 72 50       	movups 0x50(%rdx),%xmm14
   1407e5715:	45 0f 11 71 50       	movups %xmm14,0x50(%r9)
   1407e571a:	44 0f 10 72 60       	movups 0x60(%rdx),%xmm14
   1407e571f:	45 0f 11 71 60       	movups %xmm14,0x60(%r9)
   1407e5724:	44 0f 10 72 70       	movups 0x70(%rdx),%xmm14
   1407e5729:	45 0f 11 71 70       	movups %xmm14,0x70(%r9)
   1407e572e:	44 0f 10 b2 80 00 00 	movups 0x80(%rdx),%xmm14
   1407e5735:	00 
   1407e5736:	45 0f 11 b1 80 00 00 	movups %xmm14,0x80(%r9)
   1407e573d:	00 
   1407e573e:	44 0f 10 b2 90 00 00 	movups 0x90(%rdx),%xmm14
   1407e5745:	00 
   1407e5746:	45 0f 11 b1 90 00 00 	movups %xmm14,0x90(%r9)
   1407e574d:	00 
   1407e574e:	44 0f 10 b2 a0 00 00 	movups 0xa0(%rdx),%xmm14
   1407e5755:	00 
   1407e5756:	45 0f 11 b1 a0 00 00 	movups %xmm14,0xa0(%r9)
   1407e575d:	00 
   1407e575e:	44 0f 10 b2 a8 00 00 	movups 0xa8(%rdx),%xmm14
   1407e5765:	00 
   1407e5766:	45 0f 11 b1 a8 00 00 	movups %xmm14,0xa8(%r9)
   1407e576d:	00 
   1407e576e:	48 39 cf             	cmp    %rcx,%rdi
   1407e5771:	0f 85 40 ff ff ff    	jne    0x1407e56b7
   1407e5777:	48 89 b4 24 88 01 00 	mov    %rsi,0x188(%rsp)
   1407e577e:	00 
   1407e577f:	48 89 94 24 70 04 00 	mov    %rdx,0x470(%rsp)
   1407e5786:	00 
   1407e5787:	4c 89 c0             	mov    %r8,%rax
   1407e578a:	48 8b 9c 24 30 04 00 	mov    0x430(%rsp),%rbx
   1407e5791:	00 
   1407e5792:	e8 e9 8e 82 ff       	call   0x14000e680 # runtime.memequal
   1407e5797:	84 c0                	test   %al,%al
   1407e5799:	75 2a                	jne    0x1407e57c5
   1407e579b:	48 8b 84 24 28 04 00 	mov    0x428(%rsp),%rax
   1407e57a2:	00 
   1407e57a3:	48 8b 94 24 70 04 00 	mov    0x470(%rsp),%rdx
   1407e57aa:	00 
   1407e57ab:	48 8b b4 24 88 01 00 	mov    0x188(%rsp),%rsi
   1407e57b2:	00 
   1407e57b3:	48 8b bc 24 68 01 00 	mov    0x168(%rsp),%rdi
   1407e57ba:	00 
   1407e57bb:	0f 1f 44 00 00       	nopl   0x0(%rax,%rax,1)
   1407e57c0:	e9 f2 fe ff ff       	jmp    0x1407e56b7
   1407e57c5:	48 8b 84 24 78 04 00 	mov    0x478(%rsp),%rax
   1407e57cc:	00 
   1407e57cd:	e8 0e 3b 8b ff       	call   0x1400992e0 # sync.(*Mutex).Unlock
   1407e57d2:	48 8b 84 24 90 06 00 	mov    0x690(%rsp),%rax
   1407e57d9:	00 
   1407e57da:	48 8d 1d a2 d0 09 00 	lea    0x9d0a2(%rip),%rbx        # 0x140882883
   1407e57e1:	b9 05 00 00 00       	mov    $0x5,%ecx
   1407e57e6:	e8 35 c7 b9 ff       	call   0x140381f20 # net/http.(*Request).PathValue
   1407e57eb:	48 89 84 24 68 04 00 	mov    %rax,0x468(%rsp)
   1407e57f2:	00 
   1407e57f3:	48 89 9c 24 88 01 00 	mov    %rbx,0x188(%rsp)
   1407e57fa:	00 
   1407e57fb:	48 8b 84 24 48 03 00 	mov    0x348(%rsp),%rax
   1407e5802:	00 
   1407e5803:	bb 0a 00 00 00       	mov    $0xa,%ebx
   1407e5808:	e8 d3 62 8e ff       	call   0x1400cbae0 # strconv.FormatInt
   1407e580d:	b9 09 00 00 00       	mov    $0x9,%ecx
   1407e5812:	48 8b bc 24 68 04 00 	mov    0x468(%rsp),%rdi
   1407e5819:	00 
   1407e581a:	48 8b b4 24 88 01 00 	mov    0x188(%rsp),%rsi
   1407e5821:	00 
   1407e5822:	4c 8d 05 d7 a7 09 00 	lea    0x9a7d7(%rip),%r8        # 0x140880000
   1407e5829:	41 b9 01 00 00 00    	mov    $0x1,%r9d
   1407e582f:	49 89 c2             	mov    %rax,%r10
   1407e5832:	49 89 db             	mov    %rbx,%r11
   1407e5835:	31 c0                	xor    %eax,%eax
   1407e5837:	48 8d 1d 33 53 0a 00 	lea    0xa5333(%rip),%rbx        # 0x14088ab71
   1407e583e:	66 90                	xchg   %ax,%ax
   1407e5840:	e8 bb 4f 88 ff       	call   0x14006a800 # runtime.concatstring4
   1407e5845:	48 8b 8c 24 90 06 00 	mov    0x690(%rsp),%rcx
   1407e584c:	00 
   1407e584d:	48 89 c7             	mov    %rax,%rdi
   1407e5850:	48 89 de             	mov    %rbx,%rsi
   1407e5853:	41 b8 2f 01 00 00    	mov    $0x12f,%r8d
   1407e5859:	48 8b 84 24 80 06 00 	mov    0x680(%rsp),%rax
   1407e5860:	00 
   1407e5861:	48 8b 9c 24 88 06 00 	mov    0x688(%rsp),%rbx
   1407e5868:	00 
   1407e5869:	e8 92 a4 ba ff       	call   0x14038fd00 # net/http.Redirect
   1407e586e:	48 81 c4 70 06 00 00 	add    $0x670,%rsp
   1407e5875:	5d                   	pop    %rbp
   1407e5876:	c3                   	ret
   1407e5877:	e8 e4 6f fa ff       	call   0x14078c860 # main.(*Host).snapshot
   1407e587c:	48 8b 8c 24 08 01 00 	mov    0x108(%rsp),%rcx
   1407e5883:	00 
   1407e5884:	48 8b 94 24 10 01 00 	mov    0x110(%rsp),%rdx
   1407e588b:	00 
   1407e588c:	48 8d 9c 24 e8 02 00 	lea    0x2e8(%rsp),%rbx
   1407e5893:	00 
   1407e5894:	48 89 e6             	mov    %rsp,%rsi
   1407e5897:	bf 04 00 00 00       	mov    $0x4,%edi
   1407e589c:	44 0f 10 36          	movups (%rsi),%xmm14
   1407e58a0:	44 0f 11 33          	movups %xmm14,(%rbx)
   1407e58a4:	44 0f 10 76 10       	movups 0x10(%rsi),%xmm14
   1407e58a9:	44 0f 11 73 10       	movups %xmm14,0x10(%rbx)
   1407e58ae:	44 0f 10 76 20       	movups 0x20(%rsi),%xmm14
   1407e58b3:	44 0f 11 73 20       	movups %xmm14,0x20(%rbx)
   1407e58b8:	44 0f 10 76 30       	movups 0x30(%rsi),%xmm14
   1407e58bd:	44 0f 11 73 30       	movups %xmm14,0x30(%rbx)
   1407e58c2:	48 83 c6 40          	add    $0x40,%rsi
   1407e58c6:	48 83 c3 40          	add    $0x40,%rbx
   1407e58ca:	ff cf                	dec    %edi
   1407e58cc:	75 ce                	jne    0x1407e589c
   1407e58ce:	44 0f 10 36          	movups (%rsi),%xmm14
   1407e58d2:	44 0f 11 33          	movups %xmm14,(%rbx)
   1407e58d6:	44 0f 10 76 10       	movups 0x10(%rsi),%xmm14
   1407e58db:	44 0f 11 73 10       	movups %xmm14,0x10(%rbx)
   1407e58e0:	44 0f 10 76 20       	movups 0x20(%rsi),%xmm14
   1407e58e5:	44 0f 11 73 20       	movups %xmm14,0x20(%rbx)
   1407e58ea:	44 0f 10 76 28       	movups 0x28(%rsi),%xmm14
   1407e58ef:	44 0f 11 73 28       	movups %xmm14,0x28(%rbx)
   1407e58f4:	31 db                	xor    %ebx,%ebx
   1407e58f6:	48 8b b4 24 58 01 00 	mov    0x158(%rsp),%rsi
   1407e58fd:	00 
   1407e58fe:	66 90                	xchg   %ax,%ax
   1407e5900:	eb 0a                	jmp    0x1407e590c
   1407e5902:	48 81 c1 98 01 00 00 	add    $0x198,%rcx
   1407e5909:	48 ff c3             	inc    %rbx
   1407e590c:	48 39 d3             	cmp    %rdx,%rbx
   1407e590f:	7d 0b                	jge    0x1407e591c
   1407e5911:	48 39 b1 b8 00 00 00 	cmp    %rsi,0xb8(%rcx)
   1407e5918:	75 e8                	jne    0x1407e5902
   1407e591a:	eb 07                	jmp    0x1407e5923
   1407e591c:	48 c7 c3 ff ff ff ff 	mov    $0xffffffffffffffff,%rbx
   1407e5923:	48 89 9c 24 48 01 00 	mov    %rbx,0x148(%rsp)
   1407e592a:	00 
   1407e592b:	48 85 db             	test   %rbx,%rbx
   1407e592e:	7d 0a                	jge    0x1407e593a
   1407e5930:	b8 01 00 00 00       	mov    $0x1,%eax
   1407e5935:	e9 d9 00 00 00       	jmp    0x1407e5a13
   1407e593a:	48 8b 94 24 f8 03 00 	mov    0x3f8(%rsp),%rdx
   1407e5941:	00 
   1407e5942:	48 39 d3             	cmp    %rdx,%rbx
   1407e5945:	0f 83 6d 0c 00 00    	jae    0x1407e65b8
   1407e594b:	48 8b 94 24 f0 03 00 	mov    0x3f0(%rsp),%rdx
   1407e5952:	00 
   1407e5953:	4c 69 db 98 01 00 00 	imul   $0x198,%rbx,%r11
   1407e595a:	46 0f b6 a4 1a e0 00 	movzbl 0xe0(%rdx,%r11,1),%r12d
   1407e5961:	00 00 
   1407e5963:	45 84 e4             	test   %r12b,%r12b
   1407e5966:	0f 84 a2 00 00 00    	je     0x1407e5a0e
   1407e596c:	4e 8b a4 1a 88 01 00 	mov    0x188(%rdx,%r11,1),%r12
   1407e5973:	00 
   1407e5974:	4c 8b ac 24 60 01 00 	mov    0x160(%rsp),%r13
   1407e597b:	00 
   1407e597c:	0f 1f 40 00          	nopl   0x0(%rax)
   1407e5980:	4d 39 ec             	cmp    %r13,%r12
   1407e5983:	74 0a                	je     0x1407e598f
   1407e5985:	b8 01 00 00 00       	mov    $0x1,%eax
   1407e598a:	e9 84 00 00 00       	jmp    0x1407e5a13
   1407e598f:	4e 8b a4 1a 78 01 00 	mov    0x178(%rdx,%r11,1),%r12
   1407e5996:	00 
   1407e5997:	4d 85 e4             	test   %r12,%r12
   1407e599a:	74 07                	je     0x1407e59a3
   1407e599c:	b8 01 00 00 00       	mov    $0x1,%eax
   1407e59a1:	eb 70                	jmp    0x1407e5a13
   1407e59a3:	4e 8b a4 1a 38 01 00 	mov    0x138(%rdx,%r11,1),%r12
   1407e59aa:	00 
   1407e59ab:	4d 85 e4             	test   %r12,%r12
   1407e59ae:	74 07                	je     0x1407e59b7
   1407e59b0:	b8 01 00 00 00       	mov    $0x1,%eax
   1407e59b5:	eb 5c                	jmp    0x1407e5a13
   1407e59b7:	4a 8b 84 1a 48 01 00 	mov    0x148(%rdx,%r11,1),%rax
   1407e59be:	00 
   1407e59bf:	4a 8b 9c 1a 50 01 00 	mov    0x150(%rdx,%r11,1),%rbx
   1407e59c6:	00 
   1407e59c7:	4a 8b 8c 1a 58 01 00 	mov    0x158(%rdx,%r11,1),%rcx
   1407e59ce:	00 
   1407e59cf:	4a 8b bc 1a 60 01 00 	mov    0x160(%rdx,%r11,1),%rdi
   1407e59d6:	00 
   1407e59d7:	4a 8b b4 1a 68 01 00 	mov    0x168(%rdx,%r11,1),%rsi
   1407e59de:	00 
   1407e59df:	4e 8b 84 1a 70 01 00 	mov    0x170(%rdx,%r11,1),%r8
   1407e59e6:	00 
   1407e59e7:	4e 8b 8c 1a 78 01 00 	mov    0x178(%rdx,%r11,1),%r9
   1407e59ee:	00 
   1407e59ef:	4e 8b 94 1a 80 01 00 	mov    0x180(%rdx,%r11,1),%r10
   1407e59f6:	00 
   1407e59f7:	e8 24 c7 fa ff       	call   0x140792120 # main.InspectionState.Pending
   1407e59fc:	48 8b 9c 24 48 01 00 	mov    0x148(%rsp),%rbx
   1407e5a03:	00 
   1407e5a04:	48 8b b4 24 58 01 00 	mov    0x158(%rsp),%rsi
   1407e5a0b:	00 
   1407e5a0c:	eb 05                	jmp    0x1407e5a13
   1407e5a0e:	b8 01 00 00 00       	mov    $0x1,%eax
   1407e5a13:	84 c0                	test   %al,%al
   1407e5a15:	75 72                	jne    0x1407e5a89
   1407e5a17:	48 8b 94 24 f8 03 00 	mov    0x3f8(%rsp),%rdx
   1407e5a1e:	00 
   1407e5a1f:	90                   	nop
   1407e5a20:	48 39 d3             	cmp    %rdx,%rbx
   1407e5a23:	0f 83 8a 0b 00 00    	jae    0x1407e65b3
   1407e5a29:	48 8b 94 24 f0 03 00 	mov    0x3f0(%rsp),%rdx
   1407e5a30:	00 
   1407e5a31:	4c 69 db 98 01 00 00 	imul   $0x198,%rbx,%r11
   1407e5a38:	4c 89 9c 24 88 01 00 	mov    %r11,0x188(%rsp)
   1407e5a3f:	00 
   1407e5a40:	4a 8b 84 1a 08 01 00 	mov    0x108(%rdx,%r11,1),%rax
   1407e5a47:	00 
   1407e5a48:	4a 8b 9c 1a 10 01 00 	mov    0x110(%rdx,%r11,1),%rbx
   1407e5a4f:	00 
   1407e5a50:	4a 8b 8c 1a 18 01 00 	mov    0x118(%rdx,%r11,1),%rcx
   1407e5a57:	00 
   1407e5a58:	4a 8b bc 1a 20 01 00 	mov    0x120(%rdx,%r11,1),%rdi
   1407e5a5f:	00 
   1407e5a60:	4a 8b b4 1a 28 01 00 	mov    0x128(%rdx,%r11,1),%rsi
   1407e5a67:	00 
   1407e5a68:	4e 8b 84 1a 30 01 00 	mov    0x130(%rdx,%r11,1),%r8
   1407e5a6f:	00 
   1407e5a70:	4e 8b 8c 1a 38 01 00 	mov    0x138(%rdx,%r11,1),%r9
   1407e5a77:	00 
   1407e5a78:	4e 8b 94 1a 40 01 00 	mov    0x140(%rdx,%r11,1),%r10
   1407e5a7f:	00 
   1407e5a80:	e8 9b c6 fa ff       	call   0x140792120 # main.InspectionState.Pending
   1407e5a85:	84 c0                	test   %al,%al
   1407e5a87:	74 3c                	je     0x1407e5ac5
   1407e5a89:	48 8b 84 24 78 04 00 	mov    0x478(%rsp),%rax
   1407e5a90:	00 
   1407e5a91:	e8 4a 38 8b ff       	call   0x1400992e0 # sync.(*Mutex).Unlock
   1407e5a96:	48 8b 84 24 80 06 00 	mov    0x680(%rsp),%rax
   1407e5a9d:	00 
   1407e5a9e:	48 8b 9c 24 88 06 00 	mov    0x688(%rsp),%rbx
   1407e5aa5:	00 
   1407e5aa6:	48 8d 0d 8e cd 0e 00 	lea    0xecd8e(%rip),%rcx        # 0x1408d283b
   1407e5aad:	bf 57 00 00 00       	mov    $0x57,%edi
   1407e5ab2:	be 99 01 00 00       	mov    $0x199,%esi
   1407e5ab7:	e8 24 9f ba ff       	call   0x14038f9e0 # net/http.Error
   1407e5abc:	48 81 c4 70 06 00 00 	add    $0x670,%rsp
   1407e5ac3:	5d                   	pop    %rbp
   1407e5ac4:	c3                   	ret
   1407e5ac5:	48 8b 84 24 f8 03 00 	mov    0x3f8(%rsp),%rax
   1407e5acc:	00 
   1407e5acd:	48 8b 8c 24 48 01 00 	mov    0x148(%rsp),%rcx
   1407e5ad4:	00 
   1407e5ad5:	48 39 c1             	cmp    %rax,%rcx
   1407e5ad8:	0f 83 d0 0a 00 00    	jae    0x1407e65ae
   1407e5ade:	48 8b 84 24 f0 03 00 	mov    0x3f0(%rsp),%rax
   1407e5ae5:	00 
   1407e5ae6:	48 89 84 24 60 04 00 	mov    %rax,0x460(%rsp)
   1407e5aed:	00 
   1407e5aee:	e8 ad c1 8d ff       	call   0x1400c1ca0 # time.Now
   1407e5af3:	e8 c8 63 8d ff       	call   0x1400bbec0 # time.Time.UTC
   1407e5af8:	48 8d 3d 4c e4 0b 00 	lea    0xbe44c(%rip),%rdi        # 0x1408a3f4b
   1407e5aff:	be 19 00 00 00       	mov    $0x19,%esi
   1407e5b04:	e8 d7 cc 8c ff       	call   0x1400b27e0 # time.Time.Format
   1407e5b09:	48 8b 8c 24 e8 03 00 	mov    0x3e8(%rsp),%rcx
   1407e5b10:	00 
   1407e5b11:	48 89 8c 24 78 01 00 	mov    %rcx,0x178(%rsp)
   1407e5b18:	00 
   1407e5b19:	48 8d 94 24 40 02 00 	lea    0x240(%rsp),%rdx
   1407e5b20:	00 
   1407e5b21:	48 8d 35 80 01 40 00 	lea    0x400180(%rip),%rsi        # 0x140be5ca8
   1407e5b28:	44 0f 10 36          	movups (%rsi),%xmm14
   1407e5b2c:	44 0f 11 32          	movups %xmm14,(%rdx)
   1407e5b30:	44 0f 10 76 10       	movups 0x10(%rsi),%xmm14
   1407e5b35:	44 0f 11 72 10       	movups %xmm14,0x10(%rdx)
   1407e5b3a:	44 0f 10 76 20       	movups 0x20(%rsi),%xmm14
   1407e5b3f:	44 0f 11 72 20       	movups %xmm14,0x20(%rdx)
   1407e5b44:	44 0f 10 76 30       	movups 0x30(%rsi),%xmm14
   1407e5b49:	44 0f 11 72 30       	movups %xmm14,0x30(%rdx)
   1407e5b4e:	44 0f 10 76 40       	movups 0x40(%rsi),%xmm14
   1407e5b53:	44 0f 11 72 40       	movups %xmm14,0x40(%rdx)
   1407e5b58:	44 0f 10 76 50       	movups 0x50(%rsi),%xmm14
   1407e5b5d:	44 0f 11 72 50       	movups %xmm14,0x50(%rdx)
   1407e5b62:	44 0f 10 76 60       	movups 0x60(%rsi),%xmm14
   1407e5b67:	44 0f 11 72 60       	movups %xmm14,0x60(%rdx)
   1407e5b6c:	44 0f 10 76 70       	movups 0x70(%rsi),%xmm14
   1407e5b71:	44 0f 11 72 70       	movups %xmm14,0x70(%rdx)
   1407e5b76:	44 0f 10 b6 80 00 00 	movups 0x80(%rsi),%xmm14
   1407e5b7d:	00 
   1407e5b7e:	44 0f 11 b2 80 00 00 	movups %xmm14,0x80(%rdx)
   1407e5b85:	00 
   1407e5b86:	44 0f 10 b6 90 00 00 	movups 0x90(%rsi),%xmm14
   1407e5b8d:	00 
   1407e5b8e:	44 0f 11 b2 90 00 00 	movups %xmm14,0x90(%rdx)
   1407e5b95:	00 
   1407e5b96:	44 0f 10 b6 98 00 00 	movups 0x98(%rsi),%xmm14
   1407e5b9d:	00 
   1407e5b9e:	44 0f 11 b2 98 00 00 	movups %xmm14,0x98(%rdx)
   1407e5ba5:	00 
   1407e5ba6:	48 89 8c 24 40 02 00 	mov    %rcx,0x240(%rsp)
   1407e5bad:	00 
   1407e5bae:	48 8b b4 24 58 01 00 	mov    0x158(%rsp),%rsi
   1407e5bb5:	00 
   1407e5bb6:	48 89 b4 24 98 02 00 	mov    %rsi,0x298(%rsp)
   1407e5bbd:	00 
   1407e5bbe:	48 89 8c 24 a0 02 00 	mov    %rcx,0x2a0(%rsp)
   1407e5bc5:	00 
   1407e5bc6:	48 8b b4 24 20 04 00 	mov    0x420(%rsp),%rsi
   1407e5bcd:	00 
   1407e5bce:	48 89 b4 24 a8 02 00 	mov    %rsi,0x2a8(%rsp)
   1407e5bd5:	00 
   1407e5bd6:	48 8b bc 24 50 01 00 	mov    0x150(%rsp),%rdi
   1407e5bdd:	00 
   1407e5bde:	48 89 bc 24 b0 02 00 	mov    %rdi,0x2b0(%rsp)
   1407e5be5:	00 
   1407e5be6:	4c 8b 84 24 88 01 00 	mov    0x188(%rsp),%r8
   1407e5bed:	00 
   1407e5bee:	4c 8b 8c 24 60 04 00 	mov    0x460(%rsp),%r9
   1407e5bf5:	00 
   1407e5bf6:	4f 8b 94 01 d0 00 00 	mov    0xd0(%r9,%r8,1),%r10
   1407e5bfd:	00 
   1407e5bfe:	4f 8b 9c 01 d8 00 00 	mov    0xd8(%r9,%r8,1),%r11
   1407e5c05:	00 
   1407e5c06:	4c 89 94 24 b8 02 00 	mov    %r10,0x2b8(%rsp)
   1407e5c0d:	00 
   1407e5c0e:	4c 89 9c 24 c0 02 00 	mov    %r11,0x2c0(%rsp)
   1407e5c15:	00 
   1407e5c16:	4f 8b 94 01 c0 00 00 	mov    0xc0(%r9,%r8,1),%r10
   1407e5c1d:	00 
   1407e5c1e:	4f 8b 9c 01 c8 00 00 	mov    0xc8(%r9,%r8,1),%r11
   1407e5c25:	00 
   1407e5c26:	4c 89 94 24 c8 02 00 	mov    %r10,0x2c8(%rsp)
   1407e5c2d:	00 
   1407e5c2e:	4c 89 9c 24 d0 02 00 	mov    %r11,0x2d0(%rsp)
   1407e5c35:	00 
   1407e5c36:	48 89 84 24 d8 02 00 	mov    %rax,0x2d8(%rsp)
   1407e5c3d:	00 
   1407e5c3e:	48 89 9c 24 e0 02 00 	mov    %rbx,0x2e0(%rsp)
   1407e5c45:	00 
   1407e5c46:	48 89 84 24 50 02 00 	mov    %rax,0x250(%rsp)
   1407e5c4d:	00 
   1407e5c4e:	48 89 9c 24 58 02 00 	mov    %rbx,0x258(%rsp)
   1407e5c55:	00 
   1407e5c56:	4c 8d 94 24 98 01 00 	lea    0x198(%rsp),%r10
   1407e5c5d:	00 
   1407e5c5e:	44 0f 10 32          	movups (%rdx),%xmm14
   1407e5c62:	45 0f 11 32          	movups %xmm14,(%r10)
   1407e5c66:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   1407e5c6b:	45 0f 11 72 10       	movups %xmm14,0x10(%r10)
   1407e5c70:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   1407e5c75:	45 0f 11 72 20       	movups %xmm14,0x20(%r10)
   1407e5c7a:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   1407e5c7f:	45 0f 11 72 30       	movups %xmm14,0x30(%r10)
   1407e5c84:	44 0f 10 72 40       	movups 0x40(%rdx),%xmm14
   1407e5c89:	45 0f 11 72 40       	movups %xmm14,0x40(%r10)
   1407e5c8e:	44 0f 10 72 50       	movups 0x50(%rdx),%xmm14
   1407e5c93:	45 0f 11 72 50       	movups %xmm14,0x50(%r10)
   1407e5c98:	44 0f 10 72 60       	movups 0x60(%rdx),%xmm14
   1407e5c9d:	45 0f 11 72 60       	movups %xmm14,0x60(%r10)
   1407e5ca2:	44 0f 10 72 70       	movups 0x70(%rdx),%xmm14
   1407e5ca7:	45 0f 11 72 70       	movups %xmm14,0x70(%r10)
   1407e5cac:	44 0f 10 b2 80 00 00 	movups 0x80(%rdx),%xmm14
   1407e5cb3:	00 
   1407e5cb4:	45 0f 11 b2 80 00 00 	movups %xmm14,0x80(%r10)
   1407e5cbb:	00 
   1407e5cbc:	44 0f 10 b2 90 00 00 	movups 0x90(%rdx),%xmm14
   1407e5cc3:	00 
   1407e5cc4:	45 0f 11 b2 90 00 00 	movups %xmm14,0x90(%r10)
   1407e5ccb:	00 
   1407e5ccc:	44 0f 10 b2 98 00 00 	movups 0x98(%rdx),%xmm14
   1407e5cd3:	00 
   1407e5cd4:	45 0f 11 b2 98 00 00 	movups %xmm14,0x98(%r10)
   1407e5cdb:	00 
   1407e5cdc:	4c 8d 59 01          	lea    0x1(%rcx),%r11
   1407e5ce0:	4c 89 9c 24 f8 01 00 	mov    %r11,0x1f8(%rsp)
   1407e5ce7:	00 
   1407e5ce8:	4c 8d 25 71 30 0a 00 	lea    0xa3071(%rip),%r12        # 0x140888d60
   1407e5cef:	4c 89 a4 24 e0 01 00 	mov    %r12,0x1e0(%rsp)
   1407e5cf6:	00 
   1407e5cf7:	48 c7 84 24 e8 01 00 	movq   $0x8,0x1e8(%rsp)
   1407e5cfe:	00 08 00 00 00 
   1407e5d03:	4c 8d a4 24 30 06 00 	lea    0x630(%rsp),%r12
   1407e5d0a:	00 
   1407e5d0b:	45 0f 11 3c 24       	movups %xmm15,(%r12)
   1407e5d10:	45 0f 11 7c 24 10    	movups %xmm15,0x10(%r12)
   1407e5d16:	45 0f 11 7c 24 20    	movups %xmm15,0x20(%r12)
   1407e5d1c:	45 0f 11 7c 24 30    	movups %xmm15,0x30(%r12)
   1407e5d22:	48 89 8c 24 30 06 00 	mov    %rcx,0x630(%rsp)
   1407e5d29:	00 
   1407e5d2a:	48 89 b4 24 48 06 00 	mov    %rsi,0x648(%rsp)
   1407e5d31:	00 
   1407e5d32:	48 89 bc 24 50 06 00 	mov    %rdi,0x650(%rsp)
   1407e5d39:	00 
   1407e5d3a:	48 89 84 24 38 06 00 	mov    %rax,0x638(%rsp)
   1407e5d41:	00 
   1407e5d42:	48 89 9c 24 40 06 00 	mov    %rbx,0x640(%rsp)
   1407e5d49:	00 
   1407e5d4a:	48 89 8c 24 68 06 00 	mov    %rcx,0x668(%rsp)
   1407e5d51:	00 
   1407e5d52:	4f 8d 2c 01          	lea    (%r9,%r8,1),%r13
   1407e5d56:	4d 8d ad 48 01 00 00 	lea    0x148(%r13),%r13
   1407e5d5d:	4f 8d 3c 01          	lea    (%r9,%r8,1),%r15
   1407e5d61:	4d 8d bf 08 01 00 00 	lea    0x108(%r15),%r15
   1407e5d68:	4c 89 bc 24 58 04 00 	mov    %r15,0x458(%rsp)
   1407e5d6f:	00 
   1407e5d70:	83 3d e9 fb 2a 04 00 	cmpl   $0x0,0x42afbe9(%rip)        # 0x144a95960
   1407e5d77:	0f 84 9a 00 00 00    	je     0x1407e5e17
   1407e5d7d:	4c 89 9c 24 80 01 00 	mov    %r11,0x180(%rsp)
   1407e5d84:	00 
   1407e5d85:	4c 89 ac 24 50 04 00 	mov    %r13,0x450(%rsp)
   1407e5d8c:	00 
   1407e5d8d:	48 89 9c 24 40 01 00 	mov    %rbx,0x140(%rsp)
   1407e5d94:	00 
   1407e5d95:	48 89 84 24 90 01 00 	mov    %rax,0x190(%rsp)
   1407e5d9c:	00 
   1407e5d9d:	48 8d 05 7c e9 be 00 	lea    0xbee97c(%rip),%rax        # 0x1413d4720
   1407e5da4:	4c 89 eb             	mov    %r13,%rbx
   1407e5da7:	4c 89 e1             	mov    %r12,%rcx
   1407e5daa:	e8 b1 ff 83 ff       	call   0x140025d60 # runtime.wbMove
   1407e5daf:	48 8b 84 24 90 01 00 	mov    0x190(%rsp),%rax
   1407e5db6:	00 
   1407e5db7:	48 8b 8c 24 78 01 00 	mov    0x178(%rsp),%rcx
   1407e5dbe:	00 
   1407e5dbf:	48 8d 94 24 40 02 00 	lea    0x240(%rsp),%rdx
   1407e5dc6:	00 
   1407e5dc7:	48 8b 9c 24 40 01 00 	mov    0x140(%rsp),%rbx
   1407e5dce:	00 
   1407e5dcf:	48 8b b4 24 20 04 00 	mov    0x420(%rsp),%rsi
   1407e5dd6:	00 
   1407e5dd7:	48 8b bc 24 50 01 00 	mov    0x150(%rsp),%rdi
   1407e5dde:	00 
   1407e5ddf:	4c 8b 84 24 88 01 00 	mov    0x188(%rsp),%r8
   1407e5de6:	00 
   1407e5de7:	4c 8b 8c 24 60 04 00 	mov    0x460(%rsp),%r9
   1407e5dee:	00 
   1407e5def:	4c 8d 94 24 98 01 00 	lea    0x198(%rsp),%r10
   1407e5df6:	00 
   1407e5df7:	4c 8b 9c 24 80 01 00 	mov    0x180(%rsp),%r11
   1407e5dfe:	00 
   1407e5dff:	4c 8d a4 24 30 06 00 	lea    0x630(%rsp),%r12
   1407e5e06:	00 
   1407e5e07:	4c 8b ac 24 50 04 00 	mov    0x450(%rsp),%r13
   1407e5e0e:	00 
   1407e5e0f:	4c 8b bc 24 58 04 00 	mov    0x458(%rsp),%r15
   1407e5e16:	00 
   1407e5e17:	45 0f 10 34 24       	movups (%r12),%xmm14
   1407e5e1c:	45 0f 11 75 00       	movups %xmm14,0x0(%r13)
   1407e5e21:	45 0f 10 74 24 10    	movups 0x10(%r12),%xmm14
   1407e5e27:	45 0f 11 75 10       	movups %xmm14,0x10(%r13)
   1407e5e2c:	45 0f 10 74 24 20    	movups 0x20(%r12),%xmm14
   1407e5e32:	45 0f 11 75 20       	movups %xmm14,0x20(%r13)
   1407e5e37:	45 0f 10 74 24 30    	movups 0x30(%r12),%xmm14
   1407e5e3d:	45 0f 11 75 30       	movups %xmm14,0x30(%r13)
   1407e5e42:	4c 8d a4 24 f0 05 00 	lea    0x5f0(%rsp),%r12
   1407e5e49:	00 
   1407e5e4a:	45 0f 11 3c 24       	movups %xmm15,(%r12)
   1407e5e4f:	45 0f 11 7c 24 10    	movups %xmm15,0x10(%r12)
   1407e5e55:	45 0f 11 7c 24 20    	movups %xmm15,0x20(%r12)
   1407e5e5b:	45 0f 11 7c 24 30    	movups %xmm15,0x30(%r12)
   1407e5e61:	48 89 8c 24 f0 05 00 	mov    %rcx,0x5f0(%rsp)
   1407e5e68:	00 
   1407e5e69:	48 89 b4 24 08 06 00 	mov    %rsi,0x608(%rsp)
   1407e5e70:	00 
   1407e5e71:	48 89 bc 24 10 06 00 	mov    %rdi,0x610(%rsp)
   1407e5e78:	00 
   1407e5e79:	48 89 84 24 f8 05 00 	mov    %rax,0x5f8(%rsp)
   1407e5e80:	00 
   1407e5e81:	48 89 9c 24 00 06 00 	mov    %rbx,0x600(%rsp)
   1407e5e88:	00 
   1407e5e89:	4c 89 9c 24 28 06 00 	mov    %r11,0x628(%rsp)
   1407e5e90:	00 
   1407e5e91:	83 3d c8 fa 2a 04 00 	cmpl   $0x0,0x42afac8(%rip)        # 0x144a95960
   1407e5e98:	74 4a                	je     0x1407e5ee4
   1407e5e9a:	48 8d 05 7f e8 be 00 	lea    0xbee87f(%rip),%rax        # 0x1413d4720
   1407e5ea1:	4c 89 fb             	mov    %r15,%rbx
   1407e5ea4:	4c 89 e1             	mov    %r12,%rcx
   1407e5ea7:	e8 b4 fe 83 ff       	call   0x140025d60 # runtime.wbMove
   1407e5eac:	48 8b 8c 24 78 01 00 	mov    0x178(%rsp),%rcx
   1407e5eb3:	00 
   1407e5eb4:	48 8d 94 24 40 02 00 	lea    0x240(%rsp),%rdx
   1407e5ebb:	00 
   1407e5ebc:	4c 8b 84 24 88 01 00 	mov    0x188(%rsp),%r8
   1407e5ec3:	00 
   1407e5ec4:	4c 8b 8c 24 60 04 00 	mov    0x460(%rsp),%r9
   1407e5ecb:	00 
   1407e5ecc:	4c 8d 94 24 98 01 00 	lea    0x198(%rsp),%r10
   1407e5ed3:	00 
   1407e5ed4:	4c 8d a4 24 f0 05 00 	lea    0x5f0(%rsp),%r12
   1407e5edb:	00 
   1407e5edc:	4c 8b bc 24 58 04 00 	mov    0x458(%rsp),%r15
   1407e5ee3:	00 
   1407e5ee4:	45 0f 10 34 24       	movups (%r12),%xmm14
   1407e5ee9:	45 0f 11 37          	movups %xmm14,(%r15)
   1407e5eed:	45 0f 10 74 24 10    	movups 0x10(%r12),%xmm14
   1407e5ef3:	45 0f 11 77 10       	movups %xmm14,0x10(%r15)
   1407e5ef8:	45 0f 10 74 24 20    	movups 0x20(%r12),%xmm14
   1407e5efe:	45 0f 11 77 20       	movups %xmm14,0x20(%r15)
   1407e5f03:	45 0f 10 74 24 30    	movups 0x30(%r12),%xmm14
   1407e5f09:	45 0f 11 77 30       	movups %xmm14,0x30(%r15)
   1407e5f0e:	4b ff 84 01 90 01 00 	incq   0x190(%r9,%r8,1)
   1407e5f15:	00 
   1407e5f16:	48 83 84 24 e8 03 00 	addq   $0x2,0x3e8(%rsp)
   1407e5f1d:	00 02 
   1407e5f1f:	4c 8d 84 24 38 05 00 	lea    0x538(%rsp),%r8
   1407e5f26:	00 
   1407e5f27:	45 0f 11 38          	movups %xmm15,(%r8)
   1407e5f2b:	45 0f 11 78 10       	movups %xmm15,0x10(%r8)
   1407e5f30:	45 0f 11 78 20       	movups %xmm15,0x20(%r8)
   1407e5f35:	45 0f 11 78 30       	movups %xmm15,0x30(%r8)
   1407e5f3a:	45 0f 11 78 40       	movups %xmm15,0x40(%r8)
   1407e5f3f:	45 0f 11 78 50       	movups %xmm15,0x50(%r8)
   1407e5f44:	45 0f 11 78 60       	movups %xmm15,0x60(%r8)
   1407e5f49:	45 0f 11 78 70       	movups %xmm15,0x70(%r8)
   1407e5f4e:	45 0f 11 b8 80 00 00 	movups %xmm15,0x80(%r8)
   1407e5f55:	00 
   1407e5f56:	45 0f 11 b8 90 00 00 	movups %xmm15,0x90(%r8)
   1407e5f5d:	00 
   1407e5f5e:	45 0f 11 b8 a0 00 00 	movups %xmm15,0xa0(%r8)
   1407e5f65:	00 
   1407e5f66:	45 0f 11 b8 a8 00 00 	movups %xmm15,0xa8(%r8)
   1407e5f6d:	00 
   1407e5f6e:	4c 8d 8c 24 38 05 00 	lea    0x538(%rsp),%r9
   1407e5f75:	00 
   1407e5f76:	44 0f 10 32          	movups (%rdx),%xmm14
   1407e5f7a:	45 0f 11 31          	movups %xmm14,(%r9)
   1407e5f7e:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   1407e5f83:	45 0f 11 71 10       	movups %xmm14,0x10(%r9)
   1407e5f88:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   1407e5f8d:	45 0f 11 71 20       	movups %xmm14,0x20(%r9)
   1407e5f92:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   1407e5f97:	45 0f 11 71 30       	movups %xmm14,0x30(%r9)
   1407e5f9c:	44 0f 10 72 40       	movups 0x40(%rdx),%xmm14
   1407e5fa1:	45 0f 11 71 40       	movups %xmm14,0x40(%r9)
   1407e5fa6:	44 0f 10 72 50       	movups 0x50(%rdx),%xmm14
   1407e5fab:	45 0f 11 71 50       	movups %xmm14,0x50(%r9)
   1407e5fb0:	44 0f 10 72 60       	movups 0x60(%rdx),%xmm14
   1407e5fb5:	45 0f 11 71 60       	movups %xmm14,0x60(%r9)
   1407e5fba:	44 0f 10 72 70       	movups 0x70(%rdx),%xmm14
   1407e5fbf:	45 0f 11 71 70       	movups %xmm14,0x70(%r9)
   1407e5fc4:	44 0f 10 b2 80 00 00 	movups 0x80(%rdx),%xmm14
   1407e5fcb:	00 
   1407e5fcc:	45 0f 11 b1 80 00 00 	movups %xmm14,0x80(%r9)
   1407e5fd3:	00 
   1407e5fd4:	44 0f 10 b2 90 00 00 	movups 0x90(%rdx),%xmm14
   1407e5fdb:	00 
   1407e5fdc:	45 0f 11 b1 90 00 00 	movups %xmm14,0x90(%r9)
   1407e5fe3:	00 
   1407e5fe4:	44 0f 10 b2 98 00 00 	movups 0x98(%rdx),%xmm14
   1407e5feb:	00 
   1407e5fec:	45 0f 11 b1 98 00 00 	movups %xmm14,0x98(%r9)
   1407e5ff3:	00 
   1407e5ff4:	48 8b 94 24 30 04 00 	mov    0x430(%rsp),%rdx
   1407e5ffb:	00 
   1407e5ffc:	48 89 94 24 e0 05 00 	mov    %rdx,0x5e0(%rsp)
   1407e6003:	00 
   1407e6004:	48 8b 94 24 68 01 00 	mov    0x168(%rsp),%rdx
   1407e600b:	00 
   1407e600c:	48 89 94 24 e8 05 00 	mov    %rdx,0x5e8(%rsp)
   1407e6013:	00 
   1407e6014:	48 8d 94 24 80 04 00 	lea    0x480(%rsp),%rdx
   1407e601b:	00 
   1407e601c:	44 0f 11 3a          	movups %xmm15,(%rdx)
   1407e6020:	44 0f 11 7a 10       	movups %xmm15,0x10(%rdx)
   1407e6025:	44 0f 11 7a 20       	movups %xmm15,0x20(%rdx)
   1407e602a:	44 0f 11 7a 30       	movups %xmm15,0x30(%rdx)
   1407e602f:	44 0f 11 7a 40       	movups %xmm15,0x40(%rdx)
   1407e6034:	44 0f 11 7a 50       	movups %xmm15,0x50(%rdx)
   1407e6039:	44 0f 11 7a 60       	movups %xmm15,0x60(%rdx)
   1407e603e:	44 0f 11 7a 70       	movups %xmm15,0x70(%rdx)
   1407e6043:	44 0f 11 ba 80 00 00 	movups %xmm15,0x80(%rdx)
   1407e604a:	00 
   1407e604b:	44 0f 11 ba 90 00 00 	movups %xmm15,0x90(%rdx)
   1407e6052:	00 
   1407e6053:	44 0f 11 ba a0 00 00 	movups %xmm15,0xa0(%rdx)
   1407e605a:	00 
   1407e605b:	44 0f 11 ba a8 00 00 	movups %xmm15,0xa8(%rdx)
   1407e6062:	00 
   1407e6063:	4c 8d 8c 24 80 04 00 	lea    0x480(%rsp),%r9
   1407e606a:	00 
   1407e606b:	45 0f 10 32          	movups (%r10),%xmm14
   1407e606f:	45 0f 11 31          	movups %xmm14,(%r9)
   1407e6073:	45 0f 10 72 10       	movups 0x10(%r10),%xmm14
   1407e6078:	45 0f 11 71 10       	movups %xmm14,0x10(%r9)
   1407e607d:	45 0f 10 72 20       	movups 0x20(%r10),%xmm14
   1407e6082:	45 0f 11 71 20       	movups %xmm14,0x20(%r9)
   1407e6087:	45 0f 10 72 30       	movups 0x30(%r10),%xmm14
   1407e608c:	45 0f 11 71 30       	movups %xmm14,0x30(%r9)
   1407e6091:	45 0f 10 72 40       	movups 0x40(%r10),%xmm14
   1407e6096:	45 0f 11 71 40       	movups %xmm14,0x40(%r9)
   1407e609b:	45 0f 10 72 50       	movups 0x50(%r10),%xmm14
   1407e60a0:	45 0f 11 71 50       	movups %xmm14,0x50(%r9)
   1407e60a5:	45 0f 10 72 60       	movups 0x60(%r10),%xmm14
   1407e60aa:	45 0f 11 71 60       	movups %xmm14,0x60(%r9)
   1407e60af:	45 0f 10 72 70       	movups 0x70(%r10),%xmm14
   1407e60b4:	45 0f 11 71 70       	movups %xmm14,0x70(%r9)
   1407e60b9:	45 0f 10 b2 80 00 00 	movups 0x80(%r10),%xmm14
   1407e60c0:	00 
   1407e60c1:	45 0f 11 b1 80 00 00 	movups %xmm14,0x80(%r9)
   1407e60c8:	00 
   1407e60c9:	45 0f 10 b2 90 00 00 	movups 0x90(%r10),%xmm14
   1407e60d0:	00 
   1407e60d1:	45 0f 11 b1 90 00 00 	movups %xmm14,0x90(%r9)
   1407e60d8:	00 
   1407e60d9:	45 0f 10 b2 98 00 00 	movups 0x98(%r10),%xmm14
   1407e60e0:	00 
   1407e60e1:	45 0f 11 b1 98 00 00 	movups %xmm14,0x98(%r9)
   1407e60e8:	00 
   1407e60e9:	44 0f 11 bc 24 28 05 	movups %xmm15,0x528(%rsp)
   1407e60f0:	00 00 
   1407e60f2:	4c 8b 8c 24 18 04 00 	mov    0x418(%rsp),%r9
   1407e60f9:	00 
   1407e60fa:	48 8b 9c 24 10 04 00 	mov    0x410(%rsp),%rbx
   1407e6101:	00 
   1407e6102:	48 83 c3 02          	add    $0x2,%rbx
   1407e6106:	48 8b 84 24 08 04 00 	mov    0x408(%rsp),%rax
   1407e610d:	00 
   1407e610e:	49 39 d9             	cmp    %rbx,%r9
   1407e6111:	73 3c                	jae    0x1407e614f
   1407e6113:	4c 89 c9             	mov    %r9,%rcx
   1407e6116:	bf 02 00 00 00       	mov    $0x2,%edi
   1407e611b:	48 8d 35 36 42 bd 00 	lea    0xbd4236(%rip),%rsi        # 0x1413ba358
   1407e6122:	e8 d9 2b 8a ff       	call   0x140088d00 # runtime.growslice
   1407e6127:	48 89 8c 24 18 04 00 	mov    %rcx,0x418(%rsp)
   1407e612e:	00 
   1407e612f:	48 89 84 24 08 04 00 	mov    %rax,0x408(%rsp)
   1407e6136:	00 
   1407e6137:	48 8b 8c 24 78 01 00 	mov    0x178(%rsp),%rcx
   1407e613e:	00 
   1407e613f:	48 8d 94 24 80 04 00 	lea    0x480(%rsp),%rdx
   1407e6146:	00 
   1407e6147:	4c 8d 84 24 38 05 00 	lea    0x538(%rsp),%r8
   1407e614e:	00 
   1407e614f:	48 89 9c 24 10 04 00 	mov    %rbx,0x410(%rsp)
   1407e6156:	00 
   1407e6157:	48 69 f3 b8 00 00 00 	imul   $0xb8,%rbx,%rsi
   1407e615e:	48 8d 1c 06          	lea    (%rsi,%rax,1),%rbx
   1407e6162:	48 8d 9b 90 fe ff ff 	lea    -0x170(%rbx),%rbx
   1407e6169:	48 8d 34 06          	lea    (%rsi,%rax,1),%rsi
   1407e616d:	48 8d b6 48 ff ff ff 	lea    -0xb8(%rsi),%rsi
   1407e6174:	83 3d e5 f7 2a 04 00 	cmpl   $0x0,0x42af7e5(%rip)        # 0x144a95960
   1407e617b:	74 5b                	je     0x1407e61d8
   1407e617d:	48 89 9c 24 48 04 00 	mov    %rbx,0x448(%rsp)
   1407e6184:	00 
   1407e6185:	48 89 b4 24 40 04 00 	mov    %rsi,0x440(%rsp)
   1407e618c:	00 
   1407e618d:	48 8d 05 c4 41 bd 00 	lea    0xbd41c4(%rip),%rax        # 0x1413ba358
   1407e6194:	4c 89 c1             	mov    %r8,%rcx
   1407e6197:	e8 c4 fb 83 ff       	call   0x140025d60 # runtime.wbMove
   1407e619c:	48 8d 05 b5 41 bd 00 	lea    0xbd41b5(%rip),%rax        # 0x1413ba358
   1407e61a3:	48 8b 9c 24 40 04 00 	mov    0x440(%rsp),%rbx
   1407e61aa:	00 
   1407e61ab:	48 8d 8c 24 80 04 00 	lea    0x480(%rsp),%rcx
   1407e61b2:	00 
   1407e61b3:	e8 a8 fb 83 ff       	call   0x140025d60 # runtime.wbMove
   1407e61b8:	48 8d 94 24 80 04 00 	lea    0x480(%rsp),%rdx
   1407e61bf:	00 
   1407e61c0:	48 8b 9c 24 48 04 00 	mov    0x448(%rsp),%rbx
   1407e61c7:	00 
   1407e61c8:	48 8b b4 24 40 04 00 	mov    0x440(%rsp),%rsi
   1407e61cf:	00 
   1407e61d0:	4c 8d 84 24 38 05 00 	lea    0x538(%rsp),%r8
   1407e61d7:	00 
   1407e61d8:	45 0f 10 30          	movups (%r8),%xmm14
   1407e61dc:	44 0f 11 33          	movups %xmm14,(%rbx)
   1407e61e0:	45 0f 10 70 10       	movups 0x10(%r8),%xmm14
   1407e61e5:	44 0f 11 73 10       	movups %xmm14,0x10(%rbx)
   1407e61ea:	45 0f 10 70 20       	movups 0x20(%r8),%xmm14
   1407e61ef:	44 0f 11 73 20       	movups %xmm14,0x20(%rbx)
   1407e61f4:	45 0f 10 70 30       	movups 0x30(%r8),%xmm14
   1407e61f9:	44 0f 11 73 30       	movups %xmm14,0x30(%rbx)
   1407e61fe:	45 0f 10 70 40       	movups 0x40(%r8),%xmm14
   1407e6203:	44 0f 11 73 40       	movups %xmm14,0x40(%rbx)
   1407e6208:	45 0f 10 70 50       	movups 0x50(%r8),%xmm14
   1407e620d:	44 0f 11 73 50       	movups %xmm14,0x50(%rbx)
   1407e6212:	45 0f 10 70 60       	movups 0x60(%r8),%xmm14
   1407e6217:	44 0f 11 73 60       	movups %xmm14,0x60(%rbx)
   1407e621c:	45 0f 10 70 70       	movups 0x70(%r8),%xmm14
   1407e6221:	44 0f 11 73 70       	movups %xmm14,0x70(%rbx)
   1407e6226:	45 0f 10 b0 80 00 00 	movups 0x80(%r8),%xmm14
   1407e622d:	00 
   1407e622e:	44 0f 11 b3 80 00 00 	movups %xmm14,0x80(%rbx)
   1407e6235:	00 
   1407e6236:	45 0f 10 b0 90 00 00 	movups 0x90(%r8),%xmm14
   1407e623d:	00 
   1407e623e:	44 0f 11 b3 90 00 00 	movups %xmm14,0x90(%rbx)
   1407e6245:	00 
   1407e6246:	45 0f 10 b0 a0 00 00 	movups 0xa0(%r8),%xmm14
   1407e624d:	00 
   1407e624e:	44 0f 11 b3 a0 00 00 	movups %xmm14,0xa0(%rbx)
   1407e6255:	00 
   1407e6256:	45 0f 10 b0 a8 00 00 	movups 0xa8(%r8),%xmm14
   1407e625d:	00 
   1407e625e:	44 0f 11 b3 a8 00 00 	movups %xmm14,0xa8(%rbx)
   1407e6265:	00 
   1407e6266:	44 0f 10 32          	movups (%rdx),%xmm14
   1407e626a:	44 0f 11 36          	movups %xmm14,(%rsi)
   1407e626e:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   1407e6273:	44 0f 11 76 10       	movups %xmm14,0x10(%rsi)
   1407e6278:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   1407e627d:	44 0f 11 76 20       	movups %xmm14,0x20(%rsi)
   1407e6282:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   1407e6287:	44 0f 11 76 30       	movups %xmm14,0x30(%rsi)
   1407e628c:	44 0f 10 72 40       	movups 0x40(%rdx),%xmm14
   1407e6291:	44 0f 11 76 40       	movups %xmm14,0x40(%rsi)
   1407e6296:	44 0f 10 72 50       	movups 0x50(%rdx),%xmm14
   1407e629b:	44 0f 11 76 50       	movups %xmm14,0x50(%rsi)
   1407e62a0:	44 0f 10 72 60       	movups 0x60(%rdx),%xmm14
   1407e62a5:	44 0f 11 76 60       	movups %xmm14,0x60(%rsi)
   1407e62aa:	44 0f 10 72 70       	movups 0x70(%rdx),%xmm14
   1407e62af:	44 0f 11 76 70       	movups %xmm14,0x70(%rsi)
   1407e62b4:	44 0f 10 b2 80 00 00 	movups 0x80(%rdx),%xmm14
   1407e62bb:	00 
   1407e62bc:	44 0f 11 b6 80 00 00 	movups %xmm14,0x80(%rsi)
   1407e62c3:	00 
   1407e62c4:	44 0f 10 b2 90 00 00 	movups 0x90(%rdx),%xmm14
   1407e62cb:	00 
   1407e62cc:	44 0f 11 b6 90 00 00 	movups %xmm14,0x90(%rsi)
   1407e62d3:	00 
   1407e62d4:	44 0f 10 b2 a0 00 00 	movups 0xa0(%rdx),%xmm14
   1407e62db:	00 
   1407e62dc:	44 0f 11 b6 a0 00 00 	movups %xmm14,0xa0(%rsi)
   1407e62e3:	00 
   1407e62e4:	44 0f 10 b2 a8 00 00 	movups 0xa8(%rdx),%xmm14
   1407e62eb:	00 
   1407e62ec:	44 0f 11 b6 a8 00 00 	movups %xmm14,0xa8(%rsi)
   1407e62f3:	00 
   1407e62f4:	48 89 e1             	mov    %rsp,%rcx
   1407e62f7:	48 8d 94 24 e8 02 00 	lea    0x2e8(%rsp),%rdx
   1407e62fe:	00 
   1407e62ff:	bb 04 00 00 00       	mov    $0x4,%ebx
   1407e6304:	44 0f 10 32          	movups (%rdx),%xmm14
   1407e6308:	44 0f 11 31          	movups %xmm14,(%rcx)
   1407e630c:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   1407e6311:	44 0f 11 71 10       	movups %xmm14,0x10(%rcx)
   1407e6316:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   1407e631b:	44 0f 11 71 20       	movups %xmm14,0x20(%rcx)
   1407e6320:	44 0f 10 72 30       	movups 0x30(%rdx),%xmm14
   1407e6325:	44 0f 11 71 30       	movups %xmm14,0x30(%rcx)
   1407e632a:	48 83 c2 40          	add    $0x40,%rdx
   1407e632e:	48 83 c1 40          	add    $0x40,%rcx
   1407e6332:	ff cb                	dec    %ebx
   1407e6334:	75 ce                	jne    0x1407e6304
   1407e6336:	44 0f 10 32          	movups (%rdx),%xmm14
   1407e633a:	44 0f 11 31          	movups %xmm14,(%rcx)
   1407e633e:	44 0f 10 72 10       	movups 0x10(%rdx),%xmm14
   1407e6343:	44 0f 11 71 10       	movups %xmm14,0x10(%rcx)
   1407e6348:	44 0f 10 72 20       	movups 0x20(%rdx),%xmm14
   1407e634d:	44 0f 11 71 20       	movups %xmm14,0x20(%rcx)
   1407e6352:	44 0f 10 72 28       	movups 0x28(%rdx),%xmm14
   1407e6357:	44 0f 11 71 28       	movups %xmm14,0x28(%rcx)
   1407e635c:	48 8b 84 24 28 04 00 	mov    0x428(%rsp),%rax
   1407e6363:	00 
   1407e6364:	e8 f7 60 fa ff       	call   0x14078c460 # main.(*Host).commit
   1407e6369:	48 85 c0             	test   %rax,%rax
   1407e636c:	0f 85 ff 01 00 00    	jne    0x1407e6571
   1407e6372:	48 8b 94 24 28 04 00 	mov    0x428(%rsp),%rdx
   1407e6379:	00 
   1407e637a:	48 8b 8a 48 02 00 00 	mov    0x248(%rdx),%rcx
   1407e6381:	48 8b 9a 40 02 00 00 	mov    0x240(%rdx),%rbx
   1407e6388:	48 ff c3             	inc    %rbx
   1407e638b:	48 8b 82 38 02 00 00 	mov    0x238(%rdx),%rax
   1407e6392:	48 39 d9             	cmp    %rbx,%rcx
   1407e6395:	73 45                	jae    0x1407e63dc
   1407e6397:	bf 01 00 00 00       	mov    $0x1,%edi
   1407e639c:	48 8d 35 75 44 c1 00 	lea    0xc14475(%rip),%rsi        # 0x1413fa818
   1407e63a3:	e8 58 29 8a ff       	call   0x140088d00 # runtime.growslice
   1407e63a8:	48 8b 94 24 28 04 00 	mov    0x428(%rsp),%rdx
   1407e63af:	00 
   1407e63b0:	48 89 8a 48 02 00 00 	mov    %rcx,0x248(%rdx)
   1407e63b7:	83 3d a2 f5 2a 04 00 	cmpl   $0x0,0x42af5a2(%rip)        # 0x144a95960
   1407e63be:	66 90                	xchg   %ax,%ax
   1407e63c0:	74 13                	je     0x1407e63d5
   1407e63c2:	48 8b b2 38 02 00 00 	mov    0x238(%rdx),%rsi
   1407e63c9:	e8 12 7a 8a ff       	call   0x14008dde0 # runtime.gcWriteBarrier2
   1407e63ce:	49 89 03             	mov    %rax,(%r11)
   1407e63d1:	49 89 73 08          	mov    %rsi,0x8(%r11)
   1407e63d5:	48 89 82 38 02 00 00 	mov    %rax,0x238(%rdx)
   1407e63dc:	48 89 9a 40 02 00 00 	mov    %rbx,0x240(%rdx)
   1407e63e3:	48 69 f3 a8 00 00 00 	imul   $0xa8,%rbx,%rsi
   1407e63ea:	48 8d 1c 06          	lea    (%rsi,%rax,1),%rbx
   1407e63ee:	48 8d 9b 58 ff ff ff 	lea    -0xa8(%rbx),%rbx
   1407e63f5:	83 3d 64 f5 2a 04 00 	cmpl   $0x0,0x42af564(%rip)        # 0x144a95960
   1407e63fc:	74 24                	je     0x1407e6422
   1407e63fe:	48 89 9c 24 38 04 00 	mov    %rbx,0x438(%rsp)
   1407e6405:	00 
   1407e6406:	48 8d 05 0b 44 c1 00 	lea    0xc1440b(%rip),%rax        # 0x1413fa818
   1407e640d:	48 8d 8c 24 40 02 00 	lea    0x240(%rsp),%rcx
   1407e6414:	00 
   1407e6415:	e8 46 f9 83 ff       	call   0x140025d60 # runtime.wbMove
   1407e641a:	48 8b 9c 24 38 04 00 	mov    0x438(%rsp),%rbx
   1407e6421:	00 
   1407e6422:	48 8d 8c 24 40 02 00 	lea    0x240(%rsp),%rcx
   1407e6429:	00 
   1407e642a:	44 0f 10 31          	movups (%rcx),%xmm14
   1407e642e:	44 0f 11 33          	movups %xmm14,(%rbx)
   1407e6432:	44 0f 10 71 10       	movups 0x10(%rcx),%xmm14
   1407e6437:	44 0f 11 73 10       	movups %xmm14,0x10(%rbx)
   1407e643c:	44 0f 10 71 20       	movups 0x20(%rcx),%xmm14
   1407e6441:	44 0f 11 73 20       	movups %xmm14,0x20(%rbx)
   1407e6446:	44 0f 10 71 30       	movups 0x30(%rcx),%xmm14
   1407e644b:	44 0f 11 73 30       	movups %xmm14,0x30(%rbx)
   1407e6450:	44 0f 10 71 40       	movups 0x40(%rcx),%xmm14
   1407e6455:	44 0f 11 73 40       	movups %xmm14,0x40(%rbx)
   1407e645a:	44 0f 10 71 50       	movups 0x50(%rcx),%xmm14
   1407e645f:	44 0f 11 73 50       	movups %xmm14,0x50(%rbx)
   1407e6464:	44 0f 10 71 60       	movups 0x60(%rcx),%xmm14
   1407e6469:	44 0f 11 73 60       	movups %xmm14,0x60(%rbx)
   1407e646e:	44 0f 10 71 70       	movups 0x70(%rcx),%xmm14
   1407e6473:	44 0f 11 73 70       	movups %xmm14,0x70(%rbx)
   1407e6478:	44 0f 10 b1 80 00 00 	movups 0x80(%rcx),%xmm14
   1407e647f:	00 
   1407e6480:	44 0f 11 b3 80 00 00 	movups %xmm14,0x80(%rbx)
   1407e6487:	00 
   1407e6488:	44 0f 10 b1 90 00 00 	movups 0x90(%rcx),%xmm14
   1407e648f:	00 
   1407e6490:	44 0f 11 b3 90 00 00 	movups %xmm14,0x90(%rbx)
   1407e6497:	00 
   1407e6498:	44 0f 10 b1 98 00 00 	movups 0x98(%rcx),%xmm14
   1407e649f:	00 
   1407e64a0:	44 0f 11 b3 98 00 00 	movups %xmm14,0x98(%rbx)
   1407e64a7:	00 
   1407e64a8:	48 8b 84 24 78 04 00 	mov    0x478(%rsp),%rax
   1407e64af:	00 
   1407e64b0:	e8 2b 2e 8b ff       	call   0x1400992e0 # sync.(*Mutex).Unlock
   1407e64b5:	48 8b 8c 24 28 04 00 	mov    0x428(%rsp),%rcx
   1407e64bc:	00 
   1407e64bd:	48 8b 91 c8 02 00 00 	mov    0x2c8(%rcx),%rdx
   1407e64c4:	48 85 d2             	test   %rdx,%rdx
   1407e64c7:	74 05                	je     0x1407e64ce
   1407e64c9:	48 8b 02             	mov    (%rdx),%rax
   1407e64cc:	ff d0                	call   *%rax
   1407e64ce:	48 8b 84 24 90 06 00 	mov    0x690(%rsp),%rax
   1407e64d5:	00 
   1407e64d6:	48 8d 1d a6 c3 09 00 	lea    0x9c3a6(%rip),%rbx        # 0x140882883
   1407e64dd:	b9 05 00 00 00       	mov    $0x5,%ecx
   1407e64e2:	e8 39 ba b9 ff       	call   0x140381f20 # net/http.(*Request).PathValue
   1407e64e7:	48 89 84 24 68 04 00 	mov    %rax,0x468(%rsp)
   1407e64ee:	00 
   1407e64ef:	48 89 9c 24 88 01 00 	mov    %rbx,0x188(%rsp)
   1407e64f6:	00 
   1407e64f7:	48 8b 84 24 78 01 00 	mov    0x178(%rsp),%rax
   1407e64fe:	00 
   1407e64ff:	bb 0a 00 00 00       	mov    $0xa,%ebx
   1407e6504:	e8 d7 55 8e ff       	call   0x1400cbae0 # strconv.FormatInt
   1407e6509:	b9 09 00 00 00       	mov    $0x9,%ecx
   1407e650e:	48 8b bc 24 68 04 00 	mov    0x468(%rsp),%rdi
   1407e6515:	00 
   1407e6516:	48 8b b4 24 88 01 00 	mov    0x188(%rsp),%rsi
   1407e651d:	00 
   1407e651e:	4c 8d 05 db 9a 09 00 	lea    0x99adb(%rip),%r8        # 0x140880000
   1407e6525:	41 b9 01 00 00 00    	mov    $0x1,%r9d
   1407e652b:	49 89 c2             	mov    %rax,%r10
   1407e652e:	49 89 db             	mov    %rbx,%r11
   1407e6531:	31 c0                	xor    %eax,%eax
   1407e6533:	48 8d 1d 37 46 0a 00 	lea    0xa4637(%rip),%rbx        # 0x14088ab71
   1407e653a:	e8 c1 42 88 ff       	call   0x14006a800 # runtime.concatstring4
   1407e653f:	48 8b 8c 24 90 06 00 	mov    0x690(%rsp),%rcx
   1407e6546:	00 
   1407e6547:	48 89 c7             	mov    %rax,%rdi
   1407e654a:	48 89 de             	mov    %rbx,%rsi
   1407e654d:	41 b8 2f 01 00 00    	mov    $0x12f,%r8d
   1407e6553:	48 8b 84 24 80 06 00 	mov    0x680(%rsp),%rax
   1407e655a:	00 
   1407e655b:	48 8b 9c 24 88 06 00 	mov    0x688(%rsp),%rbx
   1407e6562:	00 
   1407e6563:	e8 98 97 ba ff       	call   0x14038fd00 # net/http.Redirect
   1407e6568:	48 81 c4 70 06 00 00 	add    $0x670,%rsp
   1407e656f:	5d                   	pop    %rbp
   1407e6570:	c3                   	ret
   1407e6571:	48 8b 84 24 78 04 00 	mov    0x478(%rsp),%rax
   1407e6578:	00 
   1407e6579:	e8 62 2d 8b ff       	call   0x1400992e0 # sync.(*Mutex).Unlock
   1407e657e:	48 8b 84 24 80 06 00 	mov    0x680(%rsp),%rax
   1407e6585:	00 
   1407e6586:	48 8b 9c 24 88 06 00 	mov    0x688(%rsp),%rbx
   1407e658d:	00 
   1407e658e:	48 8d 0d 05 7d 0e 00 	lea    0xe7d05(%rip),%rcx        # 0x1408ce29a
   1407e6595:	bf 49 00 00 00       	mov    $0x49,%edi
   1407e659a:	be f7 01 00 00       	mov    $0x1f7,%esi
   1407e659f:	90                   	nop
   1407e65a0:	e8 3b 94 ba ff       	call   0x14038f9e0 # net/http.Error
   1407e65a5:	48 81 c4 70 06 00 00 	add    $0x670,%rsp
   1407e65ac:	5d                   	pop    %rbp
   1407e65ad:	c3                   	ret
   1407e65ae:	e8 cd 7b 8a ff       	call   0x14008e180 # runtime.panicBounds
   1407e65b3:	e8 c8 7b 8a ff       	call   0x14008e180 # runtime.panicBounds
   1407e65b8:	e8 c3 7b 8a ff       	call   0x14008e180 # runtime.panicBounds
   1407e65bd:	90                   	nop
   1407e65be:	48 89 44 24 08       	mov    %rax,0x8(%rsp)
   1407e65c3:	48 89 5c 24 10       	mov    %rbx,0x10(%rsp)
   1407e65c8:	48 89 4c 24 18       	mov    %rcx,0x18(%rsp)
   1407e65cd:	e8 ee 5c 8a ff       	call   0x14008c2c0 # runtime.morestack
   1407e65d2:	48 8b 44 24 08       	mov    0x8(%rsp),%rax
   1407e65d7:	48 8b 5c 24 10       	mov    0x10(%rsp),%rbx
   1407e65dc:	48 8b 4c 24 18       	mov    0x18(%rsp),%rcx
   1407e65e1:	e9 7a ee ff ff       	jmp    0x1407e5460
   1407e65e6:	cc                   	int3
   1407e65e7:	cc                   	int3
   1407e65e8:	cc                   	int3
   1407e65e9:	cc                   	int3
   1407e65ea:	cc                   	int3
   1407e65eb:	cc                   	int3
   1407e65ec:	cc                   	int3
   1407e65ed:	cc                   	int3
   1407e65ee:	cc                   	int3
   1407e65ef:	cc                   	int3
   1407e65f0:	cc                   	int3
   1407e65f1:	cc                   	int3
   1407e65f2:	cc                   	int3
   1407e65f3:	cc                   	int3
   1407e65f4:	cc                   	int3
   1407e65f5:	cc                   	int3
   1407e65f6:	cc                   	int3
   1407e65f7:	cc                   	int3
   1407e65f8:	cc                   	int3
   1407e65f9:	cc                   	int3
   1407e65fa:	cc                   	int3
   1407e65fb:	cc                   	int3
   1407e65fc:	cc                   	int3
   1407e65fd:	cc                   	int3
   1407e65fe:	cc                   	int3
   1407e65ff:	cc                   	int3

