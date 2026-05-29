# 快速开始

[文档首页](../README.md) | [English](../en/README.md) | [中文索引](./README.md)

## 快速开始

```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    ninja "github.com/shijl0925/gin-ninja"
    "github.com/shijl0925/gin-ninja/middleware"
    "github.com/shijl0925/gin-ninja/settings"
)

type HelloInput struct {
    Name string `query:"name" binding:"required"`
}

type HelloOutput struct {
    Message string `json:"message"`
}

func sayHello(ctx *ninja.Context, in *HelloInput) (*HelloOutput, error) {
    return &HelloOutput{Message: "Hello, " + in.Name + "!"}, nil
}

func main() {
    api := ninja.New(ninja.Config{
        Title:             "Hello API",
        Version:           "1.0.0",
        DisableGinDefault: true,
    })

    api.UseGin(
        gin.Logger(),
        gin.Recovery(),
        middleware.RequestID(),
        middleware.CORSFromConfig(settings.CORSConfig{}),
    )

    r := ninja.NewRouter("/hello", ninja.WithTags("Hello"))
    ninja.Get(r, "/", sayHello, ninja.Summary("Say hello"))
    api.AddRouter(r)

    log.Fatal(api.Run(":8080"))
}
```

启动后可访问：

- 首页：`http://localhost:8080/`
- Swagger UI：`http://localhost:8080/docs`
- OpenAPI JSON：`http://localhost:8080/openapi.json`

Swagger UI 是默认文档界面。如果希望同一个 `DocsURL` 使用 ReDoc，可以设置 `Docs: ninja.Redoc()`：

```go
api := ninja.New(ninja.Config{
    Title: "Hello API",
    Docs:  ninja.Redoc(),
})
```

如果你希望首页展示后台入口按钮，可以在 `ninja.Config` 中设置 `AdminURL`。
如果你希望保留文档 UI 路由，但在生产环境隐藏首页里的 API Docs 快捷方式，可以设置 `HideDocsShortcut: true`。

## API Controller

`Controller` 接口允许你把同一资源的全部路由组织进一个结构体，在构造时注入共享依赖（数据库、Service 层），并在所有处理器中复用——与 django-ninja 的 `APIController` 模式一致。

### 接口定义

```go
type Controller interface {
    Register(r *ninja.Router)
}
```

在 `Register` 方法里把所有端点注册到传入的 `Router`，然后通过 `api.AddController` 挂载。

### 示例

```go
import (
    ninja "github.com/shijl0925/gin-ninja"
    "github.com/shijl0925/gin-ninja/pagination"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// Book 是一个 GORM 模型。嵌入 gorm.Model 后自动获得自增主键、
// CreatedAt / UpdatedAt 时间戳、以及软删除（DeletedAt）字段。
type Book struct {
    gorm.Model
    Title  string `gorm:"not null"`
    Author string `gorm:"not null"`
}

// --- 请求 / 响应 Schema ---

type BookOut struct {
    ID     uint   `json:"id"`
    Title  string `json:"title"`
    Author string `json:"author"`
}

type ListBooksInput  struct{ pagination.PageInput }
type GetBookInput    struct{ BookID uint `path:"id" binding:"required"` }
type CreateBookInput struct {
    Title  string `json:"title"  binding:"required"`
    Author string `json:"author" binding:"required"`
}
type UpdateBookInput struct {
    BookID uint   `path:"id"     binding:"required"`
    Title  string `json:"title"  binding:"omitempty"`
    Author string `json:"author" binding:"omitempty"`
}
type DeleteBookInput struct{ BookID uint `path:"id" binding:"required"` }

// --- Controller ---

type BookController struct {
    db *gorm.DB   // 在构造时注入一次，所有处理器共享
}

// Register 将全部 CRUD 端点注册到路由器。
func (c *BookController) Register(r *ninja.Router) {
    ninja.Get(r,    "/",    c.List,   ninja.Summary("列出书籍"), ninja.Paginated[BookOut]())
    ninja.Get(r,    "/:id", c.Get,    ninja.Summary("查询书籍"))
    ninja.Post(r,   "/",    c.Create, ninja.Summary("创建书籍"))
    ninja.Put(r,    "/:id", c.Update, ninja.Summary("更新书籍"))
    ninja.Delete(r, "/:id", c.Delete, ninja.Summary("删除书籍"))
}

func (c *BookController) Get(_ *ninja.Context, in *GetBookInput) (*BookOut, error) {
    var book Book
    if err := c.db.First(&book, in.BookID).Error; err != nil {
        return nil, ninja.NotFoundError()
    }
    return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

func (c *BookController) Create(_ *ninja.Context, in *CreateBookInput) (*BookOut, error) {
    book := Book{Title: in.Title, Author: in.Author}
    c.db.Create(&book)
    return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

// Update / Delete / List 结构相同，此处省略。

// --- 挂载 ---

func main() {
    db, _ := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
    db.AutoMigrate(&Book{})

    api := ninja.New(ninja.Config{Title: "Books API", Version: "1.0.0"})

    // 路由选项（Tags、Auth、Middleware）统一在此处声明。
    // 路由注册逻辑全部在 BookController.Register 内部完成。
    api.AddController("/books", &BookController{db: db},
        ninja.WithTags("Books"),
        ninja.WithTagDescription("Books", "图书目录的 CRUD 接口"),
    )

    api.Run(":8080")
}
```

### 为什么需要依赖注入？

将依赖在构造时传入结构体字段（而非直接读取包级全局变量），有三个核心好处：

| 关注点 | 使用依赖注入 | 不使用（全局变量） |
|---|---|---|
| **单元测试** | 传入内存 DB，各测试完全隔离 | 所有测试共享同一个全局，setup/teardown 容易相互污染 |
| **多实例挂载** | 同一个 Controller 类型可以用不同 DB 挂载（主库 + 只读副本） | 全局变量只能指向一个 DB |
| **依赖可见性** | 结构体定义即文档，依赖一目了然 | 隐式全局状态，难以追踪来源 |

对于小型脚本或快速原型而言，如果可测试性不是优先级，使用包级全局变量完全没问题。

### 不使用依赖注入（包级全局变量）

如果你更倾向于把共享数据库句柄放在包级别，而不是线程化传入结构体，Controller 依然可以正常编译和工作——`Controller` 接口对处理器如何访问其状态没有任何要求。

```go
package main

import (
    "sync"

    ninja "github.com/shijl0925/gin-ninja"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

// db 在 main() 中初始化一次，所有处理器共享访问。
// Controller 结构体中不存储任何指针。
var (
    db     *gorm.DB
    dbOnce sync.Once
)

func initDB() {
    dbOnce.Do(func() {
        var err error
        db, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
        if err != nil {
            panic(err)
        }
        db.AutoMigrate(&Book{})
    })
}

// StaticBookController 没有任何字段——直接访问包级 `db` 变量。
type StaticBookController struct{}

func (c *StaticBookController) Register(r *ninja.Router) {
    ninja.Get(r,    "/",    c.List,   ninja.Summary("列出书籍"))
    ninja.Get(r,    "/:id", c.Get,    ninja.Summary("查询书籍"))
    ninja.Post(r,   "/",    c.Create, ninja.Summary("创建书籍"))
    ninja.Delete(r, "/:id", c.Delete, ninja.Summary("删除书籍"))
}

func (c *StaticBookController) Get(_ *ninja.Context, in *GetBookInput) (*BookOut, error) {
    var book Book
    if err := db.First(&book, in.BookID).Error; err != nil {
        return nil, ninja.NotFoundError()
    }
    return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

func (c *StaticBookController) Create(_ *ninja.Context, in *CreateBookInput) (*BookOut, error) {
    book := Book{Title: in.Title, Author: in.Author}
    db.Create(&book)
    return &BookOut{ID: book.ID, Title: book.Title, Author: book.Author}, nil
}

// List / Delete 结构相同，直接访问 `db`，此处省略。

func main() {
    initDB()

    api := ninja.New(ninja.Config{Title: "Books API", Version: "1.0.0"})
    api.AddController("/books", &StaticBookController{}, ninja.WithTags("Books"))
    api.Run(":8080")
}
```

> **权衡** — `StaticBookController` 的接线更简单，但孤立测试更难：替换数据库需要修改全局变量，需要各自独立 DB 的并行测试会变得难以管理。生产服务推荐使用依赖注入形式。

### 使用 `ControllerFunc` 内联 Controller

如果不需要结构体，也可以用函数适配器快速定义：

```go
api.AddController("/items", ninja.ControllerFunc(func(r *ninja.Router) {
    ninja.Get(r,  "/",    listItems,  ninja.Summary("列表"))
    ninja.Post(r, "/",    createItem, ninja.Summary("创建"))
}), ninja.WithTags("Items"))
```

### Controller vs. Router 选型参考

| 场景 | 推荐方式 |
|---|---|
| 一组路由共享同一个依赖（DB、Service） | `Controller` — 依赖在结构体构造时注入一次 |
| 独立工具路由，无共享状态 | `NewRouter` + `api.AddRouter` |
| 内联 / 测试路由，无需结构体 | `ControllerFunc` |

> 完整可运行示例见 [examples/controller](../../examples/controller/)，使用 GORM + SQLite in-memory 数据库。

## 轻量模式推荐

如果你是在做中小型 CRUD / 内部 API，建议先走最短路径，再按需加层：

- 代码结构优先参考 [examples/basic](../../examples/basic/)：`New + Router + Handler + orm.Middleware`
- 脚手架优先直接用默认 `minimal`：`gin-ninja-cli startproject mysite -module github.com/acme/mysite`
- 脚手架仍保留内置的 repo interface 分层，但对中小型 CRUD 项目依然推荐先从默认 `minimal` 起步
- 只有在你明确需要 auth / admin / 更完整基础设施时，再切到 `-template standard|auth|admin`

推荐的最小 app 目录通常是：

- `app/models.go`
- `app/schemas.go`
- `app/apis.go`
- `app/routers.go`

---

上一篇: [概览](./overview.md) | 下一篇: [项目与 CRUD 脚手架](./scaffolding.md)
