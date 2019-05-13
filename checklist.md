# RoadRunner + php7 + yii2 要点

## Memroy Usage
一方面要防止Memory Leak，一方面也要防止内存占用过多影响性能。

* 每次处理完请求调用`gc_collect_cycles()`方法
* 设定一个memory limit，超过上限worker会自动关闭。

代码大约如下
```php
public function clean($limit)
    {
        gc_collect_cycles();
        if (!isset($limit)) {
            $limit = $this->getMemoryLimit();
        } else {
            $limit = $limit * 1024;
        }
        $limit = $this->getMemoryLimit();
        $bound = $limit * .90;
        $usage = memory_get_usage(true);
        if ($usage >= $bound) {
            return true;
        }

        return false;
    }
```
## opcache
* 控制台脚本中的opcache开关与否对性能的影响有限。opcache.file_cache对控制台脚本性能有所提升。具体影响有待进一步验证。

## connections - TODO
* 在yii2 db设置中配置
    ```php
    $db['attributes'] = [\PDO::ATTR_PERSISTENT => true];
    ```
    是可以在php-fpm中实现长连接的。可能担心连接数问题，目前的项目基本都没有开启。
    在控制台中，这个配置会失效。
* 长时运行的脚本数据库链接断线重连问题 https://www.yiichina.com/topic/7296?sort=desc
* redis问题类似mysql

## http 相关 - TODO
* 跳转问题 - 已解决
* php中的$_GET, $_REQUEST 等变量没有赋值。需要我们手动赋值。- 已实现
* OPTIONS方法返回worker error, 原因需要进一步查明。 - 是由于全球变量$_SERVER未赋值，已解决
* 有的同学写的代码是直接echo的，没有用Yii2的Response Class, 因此也就不能直接被yii2-psr7-bridge支持。需要我们在调用handler的前后加上`ob_start()`和`ob_get_clean()`方法。这对速度的影响有待进一步验证。

## 文件流 - TODO
* 发送文件流的支持有问题，这个需求我们似乎没有？

## 原因尚不明
* 当开启yii2 debug时，相比fpm，控制台脚本性能下降非常显著。表现为 spl_autoload 函数耗时严重。
* 控制台脚本中的`date_default_timezone_set()`函数需要耗时3毫秒左右。

## 新项目的问题
* 有的同学写了通用的response返回方法如下
```php
/**
 * @desc 通用返回ajax请求方法
 * @param $code
 * @param string $msg
 * @param array $data
 * @param int $header
 */
public static function _end($code, $msg='', $data=array(), $header = 1)
{
    $response = Yii::$app->getResponse();
    if($header ==1){
        $response->getHeaders()->add('Content-Type', 'application/json;charset=utf-8');
    }
    $code_cut = substr($code,-3);
    $msg = !empty($msg) ? $msg : Yii::t('tips',$code_cut);
    $response->content = json_encode(array(
        "code"=>$code,
        "msg"=>$msg,
        "data"=>$data,
    ));
    \Yii::$app->end();
}
```
已重写`end()`方法.
*  worker只能返回一次结果，然后似乎就block住了？ - 已解决，原因是 调用 clean() 的时候，没有设定MemrylimitMemory Limit. 已经加上了一个20mb的默认值。

## 运行一个复杂项目并验证
* 已运行客服后台后端并验证
## 运行多个复杂项目 - TODO