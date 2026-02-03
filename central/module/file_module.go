package module

import (
	"sync-node/central/service/oss"

	"go.uber.org/fx"
)

// FileModule 文件模块，提供对象存储服务
var FileModule = fx.Module("file",
	fx.Provide(
		oss.NewOSSService,
	),
)
