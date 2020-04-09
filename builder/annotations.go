package builder

import (
	"go/ast"
	"strconv"
	"strings"

	"github.com/arneph/toph/ir"
)

func (b *builder) findIterationBounds(stmt ast.Stmt, ctx *context) (min, max int) {
	min, max = -1, -1

	for _, info := range b.findInfos(stmt, ctx) {
		if strings.HasPrefix(info, "min_iter=") {
			r, err := strconv.Atoi(info[9:])
			if err == nil {
				min = r
			}
		} else if strings.HasPrefix(info, "max_iter=") {
			r, err := strconv.Atoi(info[9:])
			if err == nil {
				max = r
			}
		}
	}
	return
}

func (b *builder) findReachabilityRequirement(stmt ast.Stmt, ctx *context) ir.ReachabilityRequirement {
	for _, info := range b.findInfos(stmt, ctx) {
		if strings.HasPrefix(info, "check=") {
			if info[6:] == "reachable" {
				return ir.Reachable
			} else if info[6:] == "unreachable" {
				return ir.Unreachable
			}
		}
	}
	return ir.NoReachabilityRequirement
}

func (b *builder) findInfos(stmt ast.Stmt, ctx *context) (infos []string) {
	for _, commentGroup := range ctx.cmap[stmt] {
		text := commentGroup.Text()
		for {
			i := strings.Index(text, "toph:")
			if i == -1 {
				break
			}
			j := strings.Index(text, "\n")
			if j == -1 {
				j = len(text)
			}
			infs := text[i+5 : j]
			text = text[j:]

			for _, info := range strings.Split(infs, ",") {
				info = strings.TrimSpace(info)
				infos = append(infos, info)
			}
		}
	}
	return
}
