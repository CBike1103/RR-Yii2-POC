# RoadRunner + php7 + yii2 性能优化

## 工具
* RoadRunner: 可以作为nginx+fpm的替代品，用Go实现，亮点是用长时运行的PHP脚本作为worker.
* wrk: 轻量级http压测工具，可以用lua编写一些复杂场景测试.
* xhprof: PHP性能追踪分析工具
× yii2-psr7-bridge: psr7到yii2的bridge，同时提供了一些将yii2作为长时运行程序时的方法。

## 过程
### 1. 确认配置
php关于fpm和cli的设置有可能是分开的。fpm的设置可以在phpinfo()中查看，cli的设置在控制台查看。

确认php的opcache已安装，并且已经打开。确认php cli的配置中opcache.enable_cli已开启。

```
php -i | grep opcache
```

### 2. 创建一个yii2 demo项目

```
composer create-project --prefer-dist yiisoft/yii2-app-basic rrdemo
```

配置nginx指向该demo的web文件夹下的index.php，然后wrk跑一遍。
```
wrk -t 4 -c 100 http://localhost:xxxx
```

编写roadrunner程序，指向demo下的yii2-psr7-bridge脚本。
```php
<?php

ini_set('display_errors', 'stderr');
error_reporting(E_ALL);

defined('YII_DEBUG') or define('YII_DEBUG', true);
defined('YII_ENV') or define('YII_ENV', 'dev');

require __DIR__ . '/vendor/autoload.php';
require __DIR__ . '/vendor/yiisoft/yii2/Yii.php';

// Roadrunner relay and PSR7 object
$relay = new \Spiral\Goridge\StreamRelay(STDIN, STDOUT);
$psr7 = new \Spiral\RoadRunner\PSR7Client(new \Spiral\RoadRunner\Worker($relay));

$config = require __DIR__ . '/config/roadrunner.php';
$application = (new \yii\Psr7\web\Application($config));

// Handle each request in a loop
while ($request = $psr7->acceptRequest()) {
    try {
        $response = $application->handle($request);
        $psr7->respond($response);
    } catch (\Throwable $e) {
        // \yii\Psr7\web\ErrorHandler should handle any exceptions
        $psr7->getWorker()->error((string)$e);
        $psr7->getWorker()->stop();
    }
    if ($application->clean()) {
        $psr7->getWorker()->stop();
        return;
    }
}
```
其中`/config/roadrunner.php`是roadrunner专用的配置文件。根据这篇[文档](https://github.com/yoozoo/yii2-psr7-bridge/blob/master/README.md)做了专门的设置。
需要引入依赖
```
composer require sqiral/roadrunner
composer require sqiral/goridge
composer require charlesportwoodii/yii2-psr7-bridge dev-master
```

wrk测试
```
wrk -t 4 -c 100 http://localhost:XXXX
```

### 3. 利用xhprof

`xhprof`是一个轻量级的分层性能测量分析器。 在数据收集阶段，它跟踪调用次数与测量数据，展示程序动态调用的弧线图。 它在报告、后期处理阶段计算了独占的性能度量，例如运行经过的时间、CPU 计算时间和内存开销。 函数性能报告可以由调用者和被调用者终止。 在数据搜集阶段 XHProf 通过调用图的循环来检测递归函数，通过赋予唯一的深度名称来避免递归调用的循环。

从 https://pecl.php.net/package/xhprof 下载。

修改 php 项目中的yii2-psr7-bridge脚本如下。

```php
<?php

ini_set('display_errors', 'stderr');
error_reporting(E_ALL);

defined('YII_DEBUG') or define('YII_DEBUG', true);
defined('YII_ENV') or define('YII_ENV', 'prod');

require __DIR__ . '/vendor/autoload.php';
require __DIR__ . '/vendor/yiisoft/yii2/Yii.php';

// Roadrunner relay and PSR7 object
$relay = new \Spiral\Goridge\StreamRelay(STDIN, STDOUT);
$psr7 = new \Spiral\RoadRunner\PSR7Client(new \Spiral\RoadRunner\Worker($relay));

$config = require __DIR__ . '/config/roadrunner.php';
$application = (new \yii\Psr7\web\Application($config));

// xhprof
$XHPROF_ROOT = '/home/chenfang/Codes/general/xhprof';
include_once $XHPROF_ROOT . "/xhprof_lib/utils/xhprof_lib.php";
include_once $XHPROF_ROOT . "/xhprof_lib/utils/xhprof_runs.php";
// save raw data for this profiler run using default
// implementation of iXHProfRuns.
$xhprof_runs = new XHProfRuns_Default();


// Handle each request in a loop
while ($request = $psr7->acceptRequest()) {
    // start profiling
    xhprof_enable();
    try {
        $response = $application->handle($request);
        $psr7->respond($response);
    } catch (\Throwable $e) {
        // \yii\Psr7\web\ErrorHandler should handle any exceptions
        $psr7->getWorker()->error((string)$e);
        $psr7->getWorker()->stop();
    }

    if ($application->clean(20)) {
        $psr7->getWorker()->stop();
        return;
    }

    $xhprof_data = xhprof_disable();
    $run_id = $xhprof_runs->save_run($xhprof_data, "xhprof_support_roadrunner");
}
```
访问网页分析问题。

