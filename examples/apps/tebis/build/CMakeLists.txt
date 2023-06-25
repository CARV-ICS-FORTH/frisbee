# File: CMakeLists.txt Author: Michalis Vardoulakis <mvard@ics.forth.gr>

cmake_minimum_required(VERSION 3.11.0)
project(kreon VERSION 1.0)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)
include(FetchContent)
include(ExternalProject)
include(cmake_utils/PreventInSouceBuilds.cmake.in)

# Set a default build type if none was specified
set(default_build_type "Release")
if(EXISTS "${CMAKE_SOURCE_DIR}/.git")
  set(default_build_type "Debug")
endif()

option(
  RUN_IWYU
  "Run include-what-you-use on the source files of the key-value store library."
  OFF)

option(DISABLE_ALL_WARNINGS "Disable all warnings to parse iwyu output." OFF)

if(RUN_IWYU STREQUAL "ON")
  file(COPY ${CMAKE_SOURCE_DIR}/scripts/CI/check-iwyu-output.py
       DESTINATION ${CMAKE_CURRENT_BINARY_DIR})
endif()

if(NOT CMAKE_BUILD_TYPE AND NOT CMAKE_CONFIGURATION_TYPES)
  message(
    STATUS
      "Setting build type to '${default_build_type}' as none was specified.")
  set(CMAKE_BUILD_TYPE
      "${default_build_type}"
      CACHE STRING "Choose the type of build." FORCE)
  # Set the possible values of build type for cmake-gui
  set_property(CACHE CMAKE_BUILD_TYPE PROPERTY STRINGS "Debug" "Release"
                                               "MinSizeRel" "RelWithDebInfo")
endif()

include(GNUInstallDirs)
include(cmake_utils/mkfs.cmake.in)
include(cmake_utils/ycsb.cmake.in)

FetchContent_Declare(
  uthash # Recommendation: Stick close to the original name.
  GIT_REPOSITORY https://github.com/troydhanson/uthash.git
  SOURCE_DIR external-deps/src/uthash # (Relative) path within in the build
                                      # directory.
)

FetchContent_Declare(
  log # Recommendation: Stick close to the original name.
  GIT_REPOSITORY https://github.com/innerout/log.c.git
  GIT_TAG 1278d9cb58e2c6b848f6cbdd62c9a01eb9ed45da)

FetchContent_Declare(
  libbloom # Recommendation: Stick close to the original name.
  GIT_REPOSITORY https://github.com/innerout/libbloom
  GIT_TAG cmake)

FetchContent_Declare(
  cmake-scripts
  GIT_REPOSITORY https://github.com/StableCoder/cmake-scripts.git
  GIT_TAG main
  GIT_PROGRESS TRUE)

# Using FetchContent_Declare results in build failed because both 'json-c' and
# 'log' have a target called 'uninstall'
ExternalProject_Add(
  cjson
  GIT_REPOSITORY https://github.com/DaveGamble/cJSON.git
  GIT_TAG v1.7.15
  SOURCE_DIR _deps/cjson-src
  BINARY_DIR _deps/cjson-build
  CMAKE_ARGS -DENABLE_CJSON_TEST=OFF -DBUILD_SHARED_LIBS=OFF
             -DCMAKE_INSTALL_PREFIX=.
  # BUILD_COMMAND is invoked in BINARY_DIR, hence the '--build .'
)
ExternalProject_Get_Property(cjson BINARY_DIR)
ExternalProject_Get_Property(cjson SOURCE_DIR)
include_directories(${SOURCE_DIR})
set(cjson_LIBRARY ${BINARY_DIR}/libcjson.a)

FetchContent_GetProperties(uthash)
FetchContent_GetProperties(log)
FetchContent_GetProperties(libbloom)
FetchContent_GetProperties(cjson)
FetchContent_GetProperties(cmake-scripts)

if(NOT uthash_POPULATED)
  FetchContent_Populate(uthash)
  FetchContent_MakeAvailable(uthash)
endif()

if(NOT log_POPULATED)
  FetchContent_Populate(log)
  add_subdirectory(${log_SOURCE_DIR} ${log_BINARY_DIR})
  include_directories(${log_SOURCE_DIR}/include)
  FetchContent_MakeAvailable(log)
endif()

if(NOT libbloom_POPULATED)
  FetchContent_Populate(libbloom)
  add_subdirectory(${libbloom_SOURCE_DIR} ${libbloom_BINARY_DIR})
  include_directories(${libbloom_SOURCE_DIR}/include)
  message(${libbloom_SOURCE_DIR}/include)
  FetchContent_MakeAvailable(libbloom)
endif()

if(NOT cmake-scripts_POPULATED)
  FetchContent_Populate(cmake-scripts)
  FetchContent_MakeAvailable(cmake-scripts)
  include(${cmake-scripts_SOURCE_DIR}/sanitizers.cmake)
endif()

# If users have zookeeper installed in a custom path they can provide it to
# continue the build. e.g.
# -DCUSTOM_ZOOKEEPER_LIB_PATH=/home/user/zookeeper/usr/local/lib e.g.
# -DCUSTOM_ZOOKEEPER_INCLUDE_PATH=/home/user/zookeeper/usr/local/include/zookeeper
if(NOT CUSTOM_ZOOKEEPER_LIB_PATH)
  set(CUSTOM_ZOOKEEPER_LIB_PATH "")
  set(CUSTOM_ZOOKEEPER_INCLUDE_PATH "")
endif()

set(Zookeeper_LIB_PATHS /usr/local/lib /opt/local/lib /usr/lib/x86_64-linux-gnu
                        ${CUSTOM_ZOOKEEPER_LIB_PATH})
find_path(
  Zookeeper_INCLUDE_DIR
  zookeeper.h
  /usr/local/include
  /usr/include
  /usr/include/zookeeper
  /usr/local/include/zookeeper
  ${CUSTOM_ZOOKEEPER_INCLUDE_PATH})

find_library(
  Zookeeper_LIBRARY
  NAMES libzookeeper_mt.so
  PATHS ${Zookeeper_LIB_PATHS})

if(NOT Zookeeper_INCLUDE_DIR OR NOT Zookeeper_LIBRARY)
  message(FATAL_ERROR "Could not find zookeeper build will fail!")
else()
  message("Zookeeper headers at ${Zookeeper_INCLUDE_DIR}")
  include_directories(${Zookeeper_INCLUDE_DIR})
endif()

include(CMakeconfig.txt)
configure_file(config.h.in config.h @ONLY)

include_directories(${CMAKE_BINARY_DIR})

if("${DISABLE_ALL_WARNINGS}" STREQUAL ON)
  string(APPEND CMAKE_C_FLAGS APPEND " -w")
  string(APPEND CMAKE_CXX_FLAGS APPEND " -w")
endif()
string(APPEND CMAKE_C_FLAGS_RELEASE " -std=gnu11 -Wall -fpic")
string(APPEND CMAKE_C_FLAGS_DEBUG " -fno-omit-frame-pointer -fpic")
string(
  APPEND
  CMAKE_C_FLAGS
  " -Wstrict-prototypes -Wall -Werror -Wextra -std=gnu11 -fpic -gdwarf-4 -DTHREADED"
)
string(APPEND CMAKE_CXX_FLAGS_RELEASE " -std=gnu++11 -Wall -fpic")
string(APPEND CMAKE_CXX_FLAGS_DEBUG " -fno-omit-frame-pointer")
string(APPEND CMAKE_CXX_FLAGS " -gdwarf-4 -DTHREADED")

set(CMAKE_LINKER_FLAGS
    "-Wl,-no-as-needed -lm -Werror -pthread -lnuma -libverbs -lrdmacm -lrt ${CMAKE_LINKER_FLAGS}"
)

add_subdirectory(kreon_rdma)
add_subdirectory(kreon_lib)
add_subdirectory(kreon_server)
add_subdirectory(YCSB-CXX)
add_subdirectory(utilities)
add_subdirectory(tests)

if(KREON_BUILD_CPACK
   AND Zookeeper_INCLUDE_DIR
   AND Zookeeper_LIBRARY)
  set(CPACK_PACKAGE_NAME ${PROJECT_NAME})
  set(CPACK_PACKAGE_VERSION ${PROJECT_VERSION})
  set(CPACK_PACKAGE_RELEASE 1)
  set(CPACK_PACKAGE_VENDOR "FORTH-ICS")
  set(CPACK_PACKAGE_DESCRIPTION_SUMMARY
      "RDMA-based distributed key-value store")
  set(CPACK_GENERATOR "RPM")
  set(CPACK_RPM_PACKAGE_LICENSE "ASL v2.0")
  set(CPACK_RPM_PACKAGE_GROUP "Applications/Databases")
  set(CPACK_RPM_PACKAGE_REQUIRES
      "numactl-libs >= 2, libibverbs, librdmacm, glibc, libgcc, libnl3, zookeeper-native >= 3"
  )
  set(CPACK_RPM_PACKAGE_DESCRIPTION
      "Kreon is a high-speed and high-efficiency distributed key-value store functioning on top of an RDMA-capable network."
  )
  set(CPACK_PACKAGING_INSTALL_PREFIX /usr)
  set(CPACK_PACKAGE_FILE_NAME
      "${CPACK_PACKAGE_NAME}-${CPACK_PACKAGE_VERSION}-${CPACK_PACKAGE_RELEASE}.${CMAKE_SYSTEM_PROCESSOR}"
  )
  include(CPack)
  if(NOT TARGET uninstall)
    configure_file(
      "${CMAKE_CURRENT_SOURCE_DIR}/cmake_utils/cmake_uninstall.cmake.in"
      "${CMAKE_CURRENT_BINARY_DIR}/cmake_uninstall.cmake" IMMEDIATE @ONLY)

    add_custom_target(
      uninstall COMMAND ${CMAKE_COMMAND} -P
                        ${CMAKE_CURRENT_BINARY_DIR}/cmake_uninstall.cmake)
  endif()

endif()
